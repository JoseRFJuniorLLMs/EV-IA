package whatsapp

import (
	"context"
	"fmt"
	"text/template"
	"bytes"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// Service implements WhatsApp messaging
type Service struct {
	provider  Provider
	templates map[string]*template.Template
	log       *zap.Logger
	fromPhone string
}

// Provider defines the WhatsApp provider interface
type Provider interface {
	SendMessage(ctx context.Context, to, body string) error
	SendTemplate(ctx context.Context, to, templateName string, params map[string]string) error
}

// Config holds WhatsApp service configuration
type Config struct {
	Provider     string // twilio, meta
	AccountSID   string // Twilio Account SID
	AuthToken    string // Twilio Auth Token
	FromPhone    string // Your WhatsApp number (with country code, e.g., +5511999999999)
	MetaToken    string // Meta Business API token
	MetaPhoneID  string // Meta Phone Number ID
}

// NewService creates a new WhatsApp service
func NewService(cfg Config, log *zap.Logger) (*Service, error) {
	var provider Provider
	var err error

	switch cfg.Provider {
	case "twilio":
		provider, err = NewTwilioProvider(cfg.AccountSID, cfg.AuthToken, cfg.FromPhone)
	case "meta":
		provider, err = NewMetaProvider(cfg.MetaToken, cfg.MetaPhoneID)
	default:
		return nil, fmt.Errorf("unknown WhatsApp provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp provider: %w", err)
	}

	s := &Service{
		provider:  provider,
		templates: make(map[string]*template.Template),
		log:       log,
		fromPhone: cfg.FromPhone,
	}

	s.loadTemplates()

	return s, nil
}

// loadTemplates loads message templates
func (s *Service) loadTemplates() {
	templates := map[string]string{
		"welcome": `Olá {{.Name}}! 👋

Bem-vindo ao SIGEC-VE! Sua conta foi criada com sucesso.

Agora você pode:
• Encontrar estações de recarga próximas
• Iniciar e monitorar suas sessões de recarga
• Fazer reservas antecipadas
• Acompanhar seus gastos

Acesse o app para começar! 🚗⚡`,

		"charging_started": `🔌 Recarga Iniciada!

Estação: {{.StationName}}
Conector: {{.ConnectorID}}
Horário: {{.StartTime}}

Você receberá uma notificação quando a recarga terminar.`,

		"charging_completed": `✅ Recarga Concluída!

Estação: {{.StationName}}
Energia: {{.EnergyKWh}} kWh
Duração: {{.Duration}}
Custo: R$ {{.Cost}}

Obrigado por usar o SIGEC-VE! ⚡`,

		"charging_80_percent": `🔋 Bateria em 80%!

Sua recarga na estação {{.StationName}} atingiu 80%.

Considere liberar o carregador para outros usuários se não precisar de carga completa.`,

		"reservation_confirmed": `📅 Reserva Confirmada!

Estação: {{.StationName}}
Data: {{.Date}}
Horário: {{.StartTime}} - {{.EndTime}}
Código: {{.ReservationCode}}

Chegue até 15 minutos após o horário reservado para não perder sua reserva.`,

		"reservation_reminder": `⏰ Lembrete de Reserva

Sua reserva começa em {{.MinutesUntil}} minutos!

Estação: {{.StationName}}
Horário: {{.StartTime}}

Não se atrase! 🚗`,

		"reservation_cancelled": `❌ Reserva Cancelada

Sua reserva na estação {{.StationName}} foi cancelada.

Motivo: {{.Reason}}

Você pode fazer uma nova reserva a qualquer momento.`,

		"payment_received": `💳 Pagamento Recebido!

Valor: R$ {{.Amount}}
Método: {{.Method}}
ID: {{.PaymentID}}

Saldo atual da carteira: R$ {{.Balance}}`,

		"low_balance": `⚠️ Saldo Baixo

Seu saldo está em R$ {{.Balance}}.

Recarregue sua carteira para continuar usando os serviços de recarga sem interrupções.`,

		"station_available": `🟢 Estação Disponível!

A estação {{.StationName}} que você estava monitorando está disponível agora!

Corra para garantir sua vaga! 🏃`,

		"trip_reminder": `🚗 Lembrete de Viagem

Sua viagem "{{.TripName}}" está programada para começar amanhã!

Origem: {{.Origin}}
Destino: {{.Destination}}
Paradas de recarga planejadas: {{.ChargingStops}}

Boa viagem! ⚡`,

		"verification_code": `🔐 Código de Verificação SIGEC-VE

Seu código é: {{.Code}}

Válido por {{.ValidMinutes}} minutos.

Não compartilhe este código com ninguém.`,
	}

	for name, content := range templates {
		tmpl, err := template.New(name).Parse(content)
		if err != nil {
			s.log.Error("Failed to parse template",
				zap.String("template", name),
				zap.Error(err),
			)
			continue
		}
		s.templates[name] = tmpl
	}
}

// SendMessage sends a plain text message
func (s *Service) SendMessage(ctx context.Context, to, message string) error {
	if err := s.provider.SendMessage(ctx, to, message); err != nil {
		s.log.Error("Failed to send WhatsApp message",
			zap.String("to", to),
			zap.Error(err),
		)
		return err
	}

	s.log.Info("WhatsApp message sent",
		zap.String("to", to),
	)

	return nil
}

// SendTemplate sends a templated message
func (s *Service) SendTemplate(ctx context.Context, to, templateName string, data map[string]interface{}) error {
	tmpl, ok := s.templates[templateName]
	if !ok {
		return fmt.Errorf("template not found: %s", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendMessage(ctx, to, buf.String())
}

// SendWelcome sends a welcome message to a new user
func (s *Service) SendWelcome(ctx context.Context, user *domain.User) error {
	if user.Phone == "" {
		return nil // Skip if no phone
	}

	return s.SendTemplate(ctx, user.Phone, "welcome", map[string]interface{}{
		"Name": user.Name,
	})
}

// SendChargingStarted sends a charging started notification
func (s *Service) SendChargingStarted(ctx context.Context, user *domain.User, tx *domain.Transaction, station *domain.ChargePoint) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "charging_started", map[string]interface{}{
		"StationName": station.Name,
		"ConnectorID": tx.ConnectorID,
		"StartTime":   tx.StartTime.Format("15:04"),
	})
}

// SendChargingCompleted sends a charging completed notification
func (s *Service) SendChargingCompleted(ctx context.Context, user *domain.User, tx *domain.Transaction, cost float64) error {
	if user.Phone == "" {
		return nil
	}

	duration := ""
	if tx.EndTime != nil {
		d := tx.EndTime.Sub(tx.StartTime)
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if hours > 0 {
			duration = fmt.Sprintf("%dh %dmin", hours, minutes)
		} else {
			duration = fmt.Sprintf("%dmin", minutes)
		}
	}

	return s.SendTemplate(ctx, user.Phone, "charging_completed", map[string]interface{}{
		"StationName": tx.ChargePointID, // Would be station name in production
		"EnergyKWh":   fmt.Sprintf("%.2f", tx.MeterStop-tx.MeterStart),
		"Duration":    duration,
		"Cost":        fmt.Sprintf("%.2f", cost),
	})
}

// SendCharging80Percent sends a notification when battery reaches 80%
func (s *Service) SendCharging80Percent(ctx context.Context, user *domain.User, stationName string) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "charging_80_percent", map[string]interface{}{
		"StationName": stationName,
	})
}

// SendReservationConfirmed sends a reservation confirmation
func (s *Service) SendReservationConfirmed(ctx context.Context, user *domain.User, reservation *domain.Reservation, stationName string) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "reservation_confirmed", map[string]interface{}{
		"StationName":     stationName,
		"Date":            reservation.StartTime.Format("02/01/2006"),
		"StartTime":       reservation.StartTime.Format("15:04"),
		"EndTime":         reservation.EndTime.Format("15:04"),
		"ReservationCode": reservation.ID[:8],
	})
}

// SendReservationReminder sends a reservation reminder
func (s *Service) SendReservationReminder(ctx context.Context, user *domain.User, reservation *domain.Reservation, stationName string, minutesUntil int) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "reservation_reminder", map[string]interface{}{
		"StationName":  stationName,
		"StartTime":    reservation.StartTime.Format("15:04"),
		"MinutesUntil": minutesUntil,
	})
}

// SendReservationCancelled sends a reservation cancellation notification
func (s *Service) SendReservationCancelled(ctx context.Context, user *domain.User, stationName, reason string) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "reservation_cancelled", map[string]interface{}{
		"StationName": stationName,
		"Reason":      reason,
	})
}

// SendPaymentReceived sends a payment received notification
func (s *Service) SendPaymentReceived(ctx context.Context, user *domain.User, payment *domain.Payment, balance float64) error {
	if user.Phone == "" {
		return nil
	}

	methodName := ""
	switch payment.Method {
	case domain.PaymentMethodCreditCard:
		methodName = "Cartão de Crédito"
	case domain.PaymentMethodDebitCard:
		methodName = "Cartão de Débito"
	case domain.PaymentMethodPix:
		methodName = "PIX"
	case domain.PaymentMethodBoleto:
		methodName = "Boleto"
	case domain.PaymentMethodWallet:
		methodName = "Carteira"
	default:
		methodName = string(payment.Method)
	}

	return s.SendTemplate(ctx, user.Phone, "payment_received", map[string]interface{}{
		"Amount":    fmt.Sprintf("%.2f", payment.Amount),
		"Method":    methodName,
		"PaymentID": payment.ID[:8],
		"Balance":   fmt.Sprintf("%.2f", balance),
	})
}

// SendLowBalance sends a low balance warning
func (s *Service) SendLowBalance(ctx context.Context, user *domain.User, balance float64) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "low_balance", map[string]interface{}{
		"Balance": fmt.Sprintf("%.2f", balance),
	})
}

// SendStationAvailable sends a notification when a monitored station becomes available
func (s *Service) SendStationAvailable(ctx context.Context, user *domain.User, stationName string) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "station_available", map[string]interface{}{
		"StationName": stationName,
	})
}

// SendTripReminder sends a trip reminder
func (s *Service) SendTripReminder(ctx context.Context, user *domain.User, trip *domain.Trip) error {
	if user.Phone == "" {
		return nil
	}

	return s.SendTemplate(ctx, user.Phone, "trip_reminder", map[string]interface{}{
		"TripName":      trip.Name,
		"Origin":        trip.Origin.City,
		"Destination":   trip.Destination.City,
		"ChargingStops": len(trip.ChargingStops),
	})
}

// SendVerificationCode sends a verification code
func (s *Service) SendVerificationCode(ctx context.Context, phone, code string, validMinutes int) error {
	return s.SendTemplate(ctx, phone, "verification_code", map[string]interface{}{
		"Code":         code,
		"ValidMinutes": validMinutes,
	})
}
