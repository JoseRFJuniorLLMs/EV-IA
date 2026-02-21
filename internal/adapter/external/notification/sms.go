package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// SMSAdapter sends SMS messages via Twilio REST API
type SMSAdapter struct {
	accountSID string
	authToken  string
	fromNumber string
	httpClient *http.Client
	log        *zap.Logger
}

// NewSMSAdapter creates a new Twilio SMS adapter
func NewSMSAdapter(accountSID, authToken, fromNumber string, log *zap.Logger) *SMSAdapter {
	return &SMSAdapter{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		httpClient: &http.Client{},
		log:        log,
	}
}

// SendSMS sends a single SMS message via Twilio
func (a *SMSAdapter) SendSMS(ctx context.Context, to, message string) error {
	if a.accountSID == "" || a.authToken == "" {
		a.log.Warn("SMS adapter not configured, skipping send", zap.String("to", to))
		return nil
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", a.accountSID)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", a.fromNumber)
	data.Set("Body", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("sms: create request: %w", err)
	}

	req.SetBasicAuth(a.accountSID, a.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.log.Error("Failed to send SMS", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("sms: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var twilioErr struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.NewDecoder(resp.Body).Decode(&twilioErr)
		a.log.Error("Twilio API error",
			zap.Int("status", resp.StatusCode),
			zap.String("message", twilioErr.Message),
			zap.Int("twilio_code", twilioErr.Code),
		)
		return fmt.Errorf("sms: twilio error %d: %s", twilioErr.Code, twilioErr.Message)
	}

	a.log.Info("SMS sent successfully", zap.String("to", to))
	return nil
}

// SendBulkSMS sends the same message to multiple recipients
func (a *SMSAdapter) SendBulkSMS(ctx context.Context, recipients []string, message string) error {
	var lastErr error
	for _, to := range recipients {
		if err := a.SendSMS(ctx, to, message); err != nil {
			a.log.Error("Failed to send bulk SMS to recipient", zap.String("to", to), zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}
