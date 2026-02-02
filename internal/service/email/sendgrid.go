package email

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridProvider implements the Provider interface using SendGrid
type SendGridProvider struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *sendgrid.Client
}

// NewSendGridProvider creates a new SendGrid provider
func NewSendGridProvider(apiKey, fromEmail, fromName string) *SendGridProvider {
	return &SendGridProvider{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		client:    sendgrid.NewSendClient(apiKey),
	}
}

// Send sends an email using SendGrid
func (p *SendGridProvider) Send(ctx context.Context, to, subject, body string, isHTML bool) error {
	from := mail.NewEmail(p.fromName, p.fromEmail)
	toEmail := mail.NewEmail("", to)

	var message *mail.SGMailV3
	if isHTML {
		message = mail.NewSingleEmail(from, subject, toEmail, "", body)
	} else {
		message = mail.NewSingleEmail(from, subject, toEmail, body, "")
	}

	response, err := p.client.SendWithContext(ctx, message)
	if err != nil {
		return fmt.Errorf("sendgrid error: %w", err)
	}

	// SendGrid returns 2xx for success
	if response.StatusCode >= 300 {
		return fmt.Errorf("sendgrid returned status %d: %s", response.StatusCode, response.Body)
	}

	return nil
}

// SendWithAttachment sends an email with an attachment using SendGrid
func (p *SendGridProvider) SendWithAttachment(ctx context.Context, to, subject, body string, isHTML bool, attachmentName string, attachmentData []byte) error {
	from := mail.NewEmail(p.fromName, p.fromEmail)
	toEmail := mail.NewEmail("", to)

	message := mail.NewV3Mail()
	message.SetFrom(from)
	message.Subject = subject

	personalization := mail.NewPersonalization()
	personalization.AddTos(toEmail)
	message.AddPersonalizations(personalization)

	if isHTML {
		message.AddContent(mail.NewContent("text/html", body))
	} else {
		message.AddContent(mail.NewContent("text/plain", body))
	}

	// Add attachment
	attachment := mail.NewAttachment()
	attachment.SetContent(string(attachmentData))
	attachment.SetFilename(attachmentName)
	attachment.SetDisposition("attachment")
	message.AddAttachment(attachment)

	response, err := p.client.SendWithContext(ctx, message)
	if err != nil {
		return fmt.Errorf("sendgrid error: %w", err)
	}

	if response.StatusCode >= 300 {
		return fmt.Errorf("sendgrid returned status %d: %s", response.StatusCode, response.Body)
	}

	return nil
}

// SendBatch sends emails to multiple recipients using SendGrid
func (p *SendGridProvider) SendBatch(ctx context.Context, recipients []string, subject, body string, isHTML bool) error {
	from := mail.NewEmail(p.fromName, p.fromEmail)

	message := mail.NewV3Mail()
	message.SetFrom(from)
	message.Subject = subject

	// Add personalizations for each recipient
	for _, recipient := range recipients {
		personalization := mail.NewPersonalization()
		personalization.AddTos(mail.NewEmail("", recipient))
		message.AddPersonalizations(personalization)
	}

	if isHTML {
		message.AddContent(mail.NewContent("text/html", body))
	} else {
		message.AddContent(mail.NewContent("text/plain", body))
	}

	response, err := p.client.SendWithContext(ctx, message)
	if err != nil {
		return fmt.Errorf("sendgrid batch error: %w", err)
	}

	if response.StatusCode >= 300 {
		return fmt.Errorf("sendgrid returned status %d: %s", response.StatusCode, response.Body)
	}

	return nil
}
