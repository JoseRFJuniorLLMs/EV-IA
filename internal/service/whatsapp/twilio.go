package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TwilioProvider implements WhatsApp messaging via Twilio
type TwilioProvider struct {
	accountSID string
	authToken  string
	fromPhone  string
	baseURL    string
	client     *http.Client
}

// TwilioMessageResponse represents Twilio API response
type TwilioMessageResponse struct {
	SID         string `json:"sid"`
	Status      string `json:"status"`
	ErrorCode   int    `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewTwilioProvider creates a new Twilio WhatsApp provider
func NewTwilioProvider(accountSID, authToken, fromPhone string) (*TwilioProvider, error) {
	if accountSID == "" || authToken == "" || fromPhone == "" {
		return nil, fmt.Errorf("accountSID, authToken, and fromPhone are required")
	}

	// Ensure phone number has whatsapp: prefix for Twilio
	if !strings.HasPrefix(fromPhone, "whatsapp:") {
		fromPhone = "whatsapp:" + fromPhone
	}

	return &TwilioProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromPhone:  fromPhone,
		baseURL:    fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s", accountSID),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendMessage sends a WhatsApp message via Twilio
func (p *TwilioProvider) SendMessage(ctx context.Context, to, body string) error {
	// Ensure phone number has whatsapp: prefix
	if !strings.HasPrefix(to, "whatsapp:") {
		to = "whatsapp:" + to
	}

	// Prepare form data
	data := url.Values{}
	data.Set("From", p.fromPhone)
	data.Set("To", to)
	data.Set("Body", body)

	// Create request
	reqURL := fmt.Sprintf("%s/Messages.json", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result TwilioMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio error: %s (code: %d)", result.ErrorMessage, result.ErrorCode)
	}

	return nil
}

// SendTemplate sends a WhatsApp template message via Twilio
// Note: Twilio uses Content Templates for pre-approved messages
func (p *TwilioProvider) SendTemplate(ctx context.Context, to, templateName string, params map[string]string) error {
	// Twilio Content API for templates
	// For now, we'll use regular messages as templates need to be pre-approved
	// In production, you would use Content Templates API

	// Build the message body from params
	body := fmt.Sprintf("Template: %s\n", templateName)
	for key, value := range params {
		body += fmt.Sprintf("%s: %s\n", key, value)
	}

	return p.SendMessage(ctx, to, body)
}

// SendMediaMessage sends a WhatsApp message with media attachment
func (p *TwilioProvider) SendMediaMessage(ctx context.Context, to, body, mediaURL string) error {
	// Ensure phone number has whatsapp: prefix
	if !strings.HasPrefix(to, "whatsapp:") {
		to = "whatsapp:" + to
	}

	// Prepare form data
	data := url.Values{}
	data.Set("From", p.fromPhone)
	data.Set("To", to)
	data.Set("Body", body)
	data.Set("MediaUrl", mediaURL)

	// Create request
	reqURL := fmt.Sprintf("%s/Messages.json", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result TwilioMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio error: %s (code: %d)", result.ErrorMessage, result.ErrorCode)
	}

	return nil
}

// GetMessageStatus retrieves the status of a sent message
func (p *TwilioProvider) GetMessageStatus(ctx context.Context, messageSID string) (string, error) {
	reqURL := fmt.Sprintf("%s/Messages/%s.json", p.baseURL, messageSID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.accountSID, p.authToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result TwilioMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Status, nil
}
