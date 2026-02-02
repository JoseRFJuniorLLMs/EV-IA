package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

const (
	pagSeguroSandboxURL    = "https://sandbox.api.pagseguro.com"
	pagSeguroProductionURL = "https://api.pagseguro.com"
)

// PagSeguroProvider implements the Provider interface for PagSeguro
type PagSeguroProvider struct {
	email     string
	token     string
	baseURL   string
	isSandbox bool
	client    *http.Client
}

// NewPagSeguroProvider creates a new PagSeguro provider
func NewPagSeguroProvider(email, token string, sandbox bool) *PagSeguroProvider {
	baseURL := pagSeguroProductionURL
	if sandbox {
		baseURL = pagSeguroSandboxURL
	}

	return &PagSeguroProvider{
		email:     email,
		token:     token,
		baseURL:   baseURL,
		isSandbox: sandbox,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *PagSeguroProvider) Name() string {
	return "pagseguro"
}

// CreatePaymentIntent creates a payment intent (checkout session)
func (p *PagSeguroProvider) CreatePaymentIntent(ctx context.Context, amount float64, currency string, metadata map[string]string) (*domain.PaymentIntent, error) {
	// PagSeguro uses checkout sessions
	reqBody := map[string]interface{}{
		"reference_id": metadata["payment_id"],
		"customer": map[string]interface{}{
			"name":  "Customer",
			"email": "customer@example.com",
		},
		"items": []map[string]interface{}{
			{
				"reference_id": "charging",
				"name":         "EV Charging",
				"quantity":     1,
				"unit_amount":  int(amount * 100),
			},
		},
		"notification_urls": []string{
			metadata["webhook_url"],
		},
	}

	resp, err := p.doRequest(ctx, "POST", "/checkouts", reqBody)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID    string `json:"id"`
		Links []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Find payment link
	var paymentURL string
	for _, link := range result.Links {
		if link.Rel == "PAY" {
			paymentURL = link.Href
			break
		}
	}

	return &domain.PaymentIntent{
		ID:           result.ID,
		ClientSecret: paymentURL, // Use as redirect URL
		Amount:       amount,
		Currency:     currency,
		Status:       "created",
	}, nil
}

// ProcessPayment processes a payment with PagSeguro
func (p *PagSeguroProvider) ProcessPayment(ctx context.Context, amount float64, currency string, paymentMethodID string, metadata map[string]string) (string, error) {
	reqBody := map[string]interface{}{
		"reference_id": metadata["payment_id"],
		"description":  "EV Charging Payment",
		"amount": map[string]interface{}{
			"value":    int(amount * 100),
			"currency": "BRL",
		},
		"payment_method": map[string]interface{}{
			"type": "CREDIT_CARD",
			"card": map[string]interface{}{
				"id": paymentMethodID,
			},
		},
	}

	resp, err := p.doRequest(ctx, "POST", "/charges", reqBody)
	if err != nil {
		return "", err
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "PAID" && result.Status != "AUTHORIZED" {
		return "", fmt.Errorf("payment not successful: %s", result.Status)
	}

	return result.ID, nil
}

// CreatePixPayment creates a PIX payment
func (p *PagSeguroProvider) CreatePixPayment(ctx context.Context, amount float64, description string, expiresIn time.Duration) (*domain.PixPayment, string, error) {
	expirationDate := time.Now().Add(expiresIn).Format(time.RFC3339)

	reqBody := map[string]interface{}{
		"reference_id": fmt.Sprintf("pix_%d", time.Now().UnixNano()),
		"description":  description,
		"amount": map[string]interface{}{
			"value":    int(amount * 100),
			"currency": "BRL",
		},
		"payment_method": map[string]interface{}{
			"type": "PIX",
			"pix": map[string]interface{}{
				"expiration_date": expirationDate,
			},
		},
	}

	resp, err := p.doRequest(ctx, "POST", "/charges", reqBody)
	if err != nil {
		return nil, "", err
	}

	var result struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PaymentMethod struct {
			Pix struct {
				QRCodes []struct {
					ID        string    `json:"id"`
					Text      string    `json:"text"`
					ExpiresAt time.Time `json:"expiration_date"`
					Links     []struct {
						Rel  string `json:"rel"`
						Href string `json:"href"`
					} `json:"links"`
				} `json:"qr_codes"`
			} `json:"pix"`
		} `json:"payment_method"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	pixPayment := &domain.PixPayment{
		ExpiresAt: time.Now().Add(expiresIn),
	}

	if len(result.PaymentMethod.Pix.QRCodes) > 0 {
		qrCode := result.PaymentMethod.Pix.QRCodes[0]
		pixPayment.CopyPaste = qrCode.Text
		pixPayment.ExpiresAt = qrCode.ExpiresAt

		// Find QR code image link
		for _, link := range qrCode.Links {
			if link.Rel == "QRCODE.PNG" {
				pixPayment.QRCodeImage = link.Href
			}
		}
	}

	return pixPayment, result.ID, nil
}

// CreateBoletoPayment creates a Boleto payment
func (p *PagSeguroProvider) CreateBoletoPayment(ctx context.Context, amount float64, customerInfo map[string]string, expiresAt time.Time) (*domain.BoletoPayment, string, error) {
	reqBody := map[string]interface{}{
		"reference_id": fmt.Sprintf("boleto_%d", time.Now().UnixNano()),
		"description":  "EV Charging - SIGEC-VE",
		"amount": map[string]interface{}{
			"value":    int(amount * 100),
			"currency": "BRL",
		},
		"payment_method": map[string]interface{}{
			"type": "BOLETO",
			"boleto": map[string]interface{}{
				"due_date": expiresAt.Format("2006-01-02"),
				"instruction_lines": map[string]string{
					"line_1": "Pagamento referente a recarga de veiculo eletrico",
					"line_2": "SIGEC-VE - Sistema de Gestao de Carregamento",
				},
				"holder": map[string]interface{}{
					"name":    customerInfo["name"],
					"tax_id":  customerInfo["cpf"],
					"email":   customerInfo["email"],
					"address": map[string]interface{}{
						"street":      customerInfo["street"],
						"number":      customerInfo["number"],
						"locality":    customerInfo["city"],
						"city":        customerInfo["city"],
						"region_code": customerInfo["state"],
						"country":     "BRA",
						"postal_code": customerInfo["postal_code"],
					},
				},
			},
		},
	}

	resp, err := p.doRequest(ctx, "POST", "/charges", reqBody)
	if err != nil {
		return nil, "", err
	}

	var result struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PaymentMethod struct {
			Boleto struct {
				ID            string `json:"id"`
				Barcode       string `json:"barcode"`
				FormattedBarcode string `json:"formatted_barcode"`
				DueDate       string `json:"due_date"`
				Links         []struct {
					Rel  string `json:"rel"`
					Href string `json:"href"`
				} `json:"links"`
			} `json:"boleto"`
		} `json:"payment_method"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	boleto := result.PaymentMethod.Boleto
	boletoPayment := &domain.BoletoPayment{
		Barcode:       boleto.Barcode,
		DigitableLine: boleto.FormattedBarcode,
		ExpiresAt:     expiresAt,
	}

	// Find boleto PDF link
	for _, link := range boleto.Links {
		if link.Rel == "PDF" {
			boletoPayment.BoletoURL = link.Href
		}
	}

	return boletoPayment, result.ID, nil
}

// RefundPayment refunds a PagSeguro payment
func (p *PagSeguroProvider) RefundPayment(ctx context.Context, paymentID string, amount float64) (string, error) {
	reqBody := map[string]interface{}{
		"amount": map[string]interface{}{
			"value": int(amount * 100),
		},
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/charges/%s/cancel", paymentID), reqBody)
	if err != nil {
		return "", err
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.ID, nil
}

// GetPayment retrieves payment details from PagSeguro
func (p *PagSeguroProvider) GetPayment(ctx context.Context, paymentID string) (*ProviderPayment, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/charges/%s", paymentID), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Value    int    `json:"value"`
			Currency string `json:"currency"`
		} `json:"amount"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := p.mapStatus(result.Status)

	return &ProviderPayment{
		ID:       result.ID,
		Status:   status,
		Amount:   float64(result.Amount.Value) / 100,
		Currency: result.Amount.Currency,
	}, nil
}

// ValidateWebhook validates PagSeguro webhook signature
func (p *PagSeguroProvider) ValidateWebhook(payload []byte, signature string) error {
	// PagSeguro uses HMAC-SHA256 for webhook validation
	mac := hmac.New(sha256.New, []byte(p.token))
	mac.Write(payload)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("invalid webhook signature")
	}

	return nil
}

// ParseWebhook parses PagSeguro webhook payload
func (p *PagSeguroProvider) ParseWebhook(payload []byte) (*WebhookEvent, error) {
	var event struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
		Charges   []struct {
			ID            string `json:"id"`
			ReferenceID   string `json:"reference_id"`
			Status        string `json:"status"`
			PaymentMethod struct {
				Type string `json:"type"`
			} `json:"payment_method"`
			Amount struct {
				Value    int    `json:"value"`
				Currency string `json:"currency"`
			} `json:"amount"`
		} `json:"charges"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	webhookEvent := &WebhookEvent{
		Type:     "payment.updated",
		Metadata: make(map[string]string),
	}

	if len(event.Charges) > 0 {
		charge := event.Charges[0]
		webhookEvent.PaymentID = charge.ID
		webhookEvent.Status = p.mapStatus(charge.Status)
		webhookEvent.Amount = float64(charge.Amount.Value) / 100
		webhookEvent.Metadata["reference_id"] = charge.ReferenceID
	}

	return webhookEvent, nil
}

// mapStatus maps PagSeguro status to domain status
func (p *PagSeguroProvider) mapStatus(status string) domain.PaymentStatus {
	switch status {
	case "PAID", "AUTHORIZED":
		return domain.PaymentStatusCompleted
	case "DECLINED", "CANCELED":
		return domain.PaymentStatusFailed
	case "IN_ANALYSIS", "WAITING":
		return domain.PaymentStatusProcessing
	default:
		return domain.PaymentStatusPending
	}
}

// doRequest performs an HTTP request to PagSeguro API
func (p *PagSeguroProvider) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("x-api-version", "4.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("PagSeguro API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
