package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPProvider implements the Provider interface using SMTP
// This is useful for development with Mailhog or other SMTP servers
type SMTPProvider struct {
	host      string
	port      int
	username  string
	password  string
	fromEmail string
	fromName  string
	useTLS    bool
}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider(host string, port int, username, password, fromEmail, fromName string, useTLS bool) *SMTPProvider {
	return &SMTPProvider{
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		fromEmail: fromEmail,
		fromName:  fromName,
		useTLS:    useTLS,
	}
}

// Send sends an email using SMTP
func (p *SMTPProvider) Send(ctx context.Context, to, subject, body string, isHTML bool) error {
	// Build email headers
	headers := make(map[string]string)
	headers["From"] = p.formatFrom()
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"

	if isHTML {
		headers["Content-Type"] = "text/html; charset=UTF-8"
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
	}

	// Build message
	var message strings.Builder
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	// Connect and send
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	if p.useTLS {
		return p.sendTLS(addr, to, message.String())
	}

	return p.sendPlain(addr, to, message.String())
}

// sendPlain sends email without TLS (for Mailhog and local development)
func (p *SMTPProvider) sendPlain(addr, to, message string) error {
	var auth smtp.Auth
	if p.username != "" && p.password != "" {
		auth = smtp.PlainAuth("", p.username, p.password, p.host)
	}

	err := smtp.SendMail(addr, auth, p.fromEmail, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("smtp error: %w", err)
	}

	return nil
}

// sendTLS sends email with TLS
func (p *SMTPProvider) sendTLS(addr, to, message string) error {
	// Connect to SMTP server
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: p.host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("tls dial error: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("smtp client error: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials provided
	if p.username != "" && p.password != "" {
		auth := smtp.PlainAuth("", p.username, p.password, p.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth error: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(p.fromEmail); err != nil {
		return fmt.Errorf("smtp mail error: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt error: %w", err)
	}

	// Send message body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data error: %w", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("smtp write error: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("smtp close error: %w", err)
	}

	return client.Quit()
}

// formatFrom formats the from address with name
func (p *SMTPProvider) formatFrom() string {
	if p.fromName != "" {
		return fmt.Sprintf("%s <%s>", p.fromName, p.fromEmail)
	}
	return p.fromEmail
}

// SendMultiple sends the same email to multiple recipients
func (p *SMTPProvider) SendMultiple(ctx context.Context, recipients []string, subject, body string, isHTML bool) error {
	for _, to := range recipients {
		if err := p.Send(ctx, to, subject, body, isHTML); err != nil {
			return fmt.Errorf("failed to send to %s: %w", to, err)
		}
	}
	return nil
}
