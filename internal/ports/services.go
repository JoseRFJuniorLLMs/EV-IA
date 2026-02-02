package ports

import (
	"context"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, string, error) // token, refresh, err
	Register(ctx context.Context, user *domain.User) error
	RefreshToken(ctx context.Context, token string) (string, error)
	ValidateToken(ctx context.Context, token string) (*domain.User, error)
}

type DeviceService interface {
	GetDevice(ctx context.Context, id string) (*domain.ChargePoint, error)
	ListDevices(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error)
	UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error
	GetNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error)
	// Voice assistant methods
	ListAvailableDevices(ctx context.Context) ([]domain.ChargePoint, error)
}

type TransactionService interface {
	StartTransaction(ctx context.Context, deviceID string, connectorID int, userID string, idTag string) (*domain.Transaction, error)
	StopTransaction(ctx context.Context, transactionID string) (*domain.Transaction, error)
	GetTransaction(ctx context.Context, id string) (*domain.Transaction, error)
	GetActiveTransaction(ctx context.Context, userID string) (*domain.Transaction, error)
	GetTransactionHistory(ctx context.Context, userID string) ([]domain.Transaction, error)
	// Voice assistant methods
	StartCharging(ctx context.Context, userID string, stationID string) (*domain.Transaction, error)
	StopActiveCharging(ctx context.Context, userID string) error
	GetCurrentSessionCost(ctx context.Context, userID string) (float64, error)
}

// BillingService handles billing and payment calculations
type BillingService interface {
	CalculateCost(ctx context.Context, tx *domain.Transaction) (float64, error)
	ProcessPayment(ctx context.Context, tx *domain.Transaction) error
	GetPricePerKWh(ctx context.Context) float64
}

// SmartChargingService handles intelligent charging optimization
type SmartChargingService interface {
	OptimizeCharging(ctx context.Context, deviceID string, targetEnergy float64) (*ChargingProfile, error)
	GetChargingProfile(ctx context.Context, deviceID string) (*ChargingProfile, error)
}

// ChargingProfile represents a smart charging schedule
type ChargingProfile struct {
	DeviceID     string
	MaxPowerKW   float64
	TargetEnergy float64
	StartTime    string
	EndTime      string
	Periods      []ChargingPeriod
}

// ChargingPeriod represents a period within a charging profile
type ChargingPeriod struct {
	StartPeriod int     // Seconds from start
	Limit       float64 // Power limit in kW
}

// EmailService handles email notifications
type EmailService interface {
	// Send sends a generic email
	Send(ctx context.Context, to, subject, body string) error

	// SendHTML sends an HTML email
	SendHTML(ctx context.Context, to, subject, htmlBody string) error

	// SendTemplate sends an email using a template
	SendTemplate(ctx context.Context, to, templateName string, data map[string]interface{}) error

	// SendWelcome sends a welcome email to a new user
	SendWelcome(ctx context.Context, user *domain.User) error

	// SendChargingStarted sends a notification when charging starts
	SendChargingStarted(ctx context.Context, user *domain.User, tx *domain.Transaction, station *domain.ChargePoint) error

	// SendChargingCompleted sends a notification when charging completes
	SendChargingCompleted(ctx context.Context, user *domain.User, tx *domain.Transaction, cost float64) error

	// SendPasswordReset sends a password reset email
	SendPasswordReset(ctx context.Context, user *domain.User, resetToken string) error

	// SendInvoice sends an invoice email
	SendInvoice(ctx context.Context, user *domain.User, invoice *Invoice) error

	// SendLowBalance sends a low balance warning
	SendLowBalance(ctx context.Context, user *domain.User, balance float64) error
}

// Invoice represents an invoice for email sending
type Invoice struct {
	ID            string
	TransactionID string
	Amount        float64
	Currency      string
	EnergyKWh     float64
	Duration      string
	StationName   string
	Date          string
}

// PaymentService handles payment processing
type PaymentService interface {
	// CreatePaymentIntent creates a payment intent for client-side confirmation
	CreatePaymentIntent(ctx context.Context, userID string, amount float64, currency string) (*domain.PaymentIntent, error)

	// ProcessPayment processes a payment
	ProcessPayment(ctx context.Context, payment *PaymentRequest) (*domain.Payment, error)

	// ProcessChargingPayment processes payment for a charging transaction
	ProcessChargingPayment(ctx context.Context, userID string, transactionID string, amount float64) (*domain.Payment, error)

	// GetPayment retrieves a payment by ID
	GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error)

	// GetPaymentHistory retrieves payment history for a user
	GetPaymentHistory(ctx context.Context, userID string, limit, offset int) ([]domain.Payment, error)

	// RefundPayment refunds a payment
	RefundPayment(ctx context.Context, paymentID string, amount float64, reason string) (*domain.Refund, error)

	// CreatePixPayment creates a PIX payment
	CreatePixPayment(ctx context.Context, userID string, amount float64) (*domain.PixPayment, *domain.Payment, error)

	// CreateBoletoPayment creates a Boleto payment
	CreateBoletoPayment(ctx context.Context, userID string, amount float64) (*domain.BoletoPayment, *domain.Payment, error)

	// HandleWebhook handles payment provider webhooks
	HandleWebhook(ctx context.Context, provider string, payload []byte, signature string) error
}

// PaymentRequest represents a payment request
type PaymentRequest struct {
	UserID        string
	Amount        float64
	Currency      string
	Method        domain.PaymentMethod
	CardID        string // For saved cards
	TransactionID string // For charging payments
	Description   string
}

// CardService handles payment card management
type CardService interface {
	// AddCard adds a new payment card
	AddCard(ctx context.Context, userID string, card *CardRequest) (*domain.PaymentCard, error)

	// GetCards retrieves all cards for a user
	GetCards(ctx context.Context, userID string) ([]domain.PaymentCard, error)

	// DeleteCard removes a card
	DeleteCard(ctx context.Context, userID string, cardID string) error

	// SetDefaultCard sets a card as default
	SetDefaultCard(ctx context.Context, userID string, cardID string) error
}

// CardRequest represents a card addition request
type CardRequest struct {
	Number      string
	ExpMonth    int
	ExpYear     int
	CVC         string
	HolderName  string
	SetDefault  bool
}

// WalletService handles user wallet operations
type WalletService interface {
	// GetWallet retrieves or creates a user's wallet
	GetWallet(ctx context.Context, userID string) (*domain.Wallet, error)

	// AddFunds adds funds to the wallet
	AddFunds(ctx context.Context, userID string, amount float64, paymentID string) error

	// DeductFunds deducts funds from the wallet
	DeductFunds(ctx context.Context, userID string, amount float64, description string, referenceID string) error

	// GetTransactions retrieves wallet transaction history
	GetTransactions(ctx context.Context, userID string, limit, offset int) ([]domain.WalletTransaction, error)

	// HasSufficientBalance checks if wallet has enough balance
	HasSufficientBalance(ctx context.Context, userID string, amount float64) (bool, error)
}

// ReservationService handles charging station reservations
type ReservationService interface {
	// CreateReservation creates a new reservation
	CreateReservation(ctx context.Context, req *ReservationRequest) (*domain.Reservation, error)

	// GetReservation retrieves a reservation by ID
	GetReservation(ctx context.Context, id string) (*domain.Reservation, error)

	// GetUserReservations retrieves all reservations for a user
	GetUserReservations(ctx context.Context, userID string, status string, limit, offset int) ([]domain.Reservation, error)

	// GetStationReservations retrieves all reservations for a station
	GetStationReservations(ctx context.Context, chargePointID string, date time.Time) ([]domain.Reservation, error)

	// CancelReservation cancels a reservation
	CancelReservation(ctx context.Context, id string, userID string, reason string) error

	// ConfirmReservation confirms a pending reservation
	ConfirmReservation(ctx context.Context, id string) error

	// ActivateReservation marks user as arrived and starts charging
	ActivateReservation(ctx context.Context, id string, transactionID string) error

	// CompleteReservation marks reservation as completed
	CompleteReservation(ctx context.Context, id string) error

	// CheckAvailability checks if a time slot is available
	CheckAvailability(ctx context.Context, chargePointID string, connectorID int, startTime, endTime time.Time) (bool, error)

	// GetAvailableSlots returns available time slots for a station
	GetAvailableSlots(ctx context.Context, chargePointID string, date time.Time) ([]domain.TimeSlot, error)

	// ProcessExpiredReservations processes reservations that have expired
	ProcessExpiredReservations(ctx context.Context) error

	// GetReservationSummary returns reservation statistics
	GetReservationSummary(ctx context.Context, chargePointID string, startDate, endDate time.Time) (*domain.ReservationSummary, error)
}

// ReservationRequest represents a reservation creation request
type ReservationRequest struct {
	UserID        string
	ChargePointID string
	ConnectorID   int
	StartTime     time.Time
	Duration      int // in minutes
	Notes         string
}

// AdminService handles administrative operations
type AdminService interface {
	// Dashboard statistics
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)
	GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (*RevenueStats, error)
	GetUsageStats(ctx context.Context, startDate, endDate time.Time) (*UsageStats, error)

	// User management
	GetUsers(ctx context.Context, filter UserFilter, limit, offset int) ([]domain.User, int, error)
	GetUserDetails(ctx context.Context, userID string) (*UserDetails, error)
	UpdateUserStatus(ctx context.Context, userID string, status string) error
	UpdateUserRole(ctx context.Context, userID string, role domain.UserRole) error

	// Station management
	GetStations(ctx context.Context, filter StationFilter, limit, offset int) ([]domain.ChargePoint, int, error)
	GetStationDetails(ctx context.Context, stationID string) (*StationDetails, error)
	UpdateStationStatus(ctx context.Context, stationID string, status domain.ChargePointStatus) error

	// Transaction management
	GetTransactions(ctx context.Context, filter TransactionFilter, limit, offset int) ([]domain.Transaction, int, error)
	GetTransactionDetails(ctx context.Context, txID string) (*TransactionDetails, error)

	// Alerts and notifications
	GetAlerts(ctx context.Context, limit, offset int) ([]Alert, error)
	AcknowledgeAlert(ctx context.Context, alertID string) error

	// Reports
	GenerateReport(ctx context.Context, reportType string, startDate, endDate time.Time) ([]byte, error)
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalUsers            int     `json:"total_users"`
	ActiveUsers           int     `json:"active_users"`
	TotalStations         int     `json:"total_stations"`
	OnlineStations        int     `json:"online_stations"`
	ActiveTransactions    int     `json:"active_transactions"`
	TodayTransactions     int     `json:"today_transactions"`
	TodayRevenue          float64 `json:"today_revenue"`
	TodayEnergyKWh        float64 `json:"today_energy_kwh"`
	PendingReservations   int     `json:"pending_reservations"`
	ActiveAlerts          int     `json:"active_alerts"`
}

// RevenueStats represents revenue statistics
type RevenueStats struct {
	TotalRevenue    float64            `json:"total_revenue"`
	RevenueByDay    map[string]float64 `json:"revenue_by_day"`
	RevenueByMethod map[string]float64 `json:"revenue_by_method"`
	AveragePerTx    float64            `json:"average_per_transaction"`
	GrowthPercent   float64            `json:"growth_percent"`
}

// UsageStats represents usage statistics
type UsageStats struct {
	TotalSessions       int                `json:"total_sessions"`
	TotalEnergyKWh      float64            `json:"total_energy_kwh"`
	AverageSessionMin   float64            `json:"average_session_minutes"`
	PeakHour            int                `json:"peak_hour"`
	SessionsByDay       map[string]int     `json:"sessions_by_day"`
	EnergyByDay         map[string]float64 `json:"energy_by_day"`
	TopStations         []StationUsage     `json:"top_stations"`
}

// StationUsage represents station usage data
type StationUsage struct {
	StationID   string  `json:"station_id"`
	StationName string  `json:"station_name"`
	Sessions    int     `json:"sessions"`
	EnergyKWh   float64 `json:"energy_kwh"`
	Revenue     float64 `json:"revenue"`
}

// UserFilter for filtering users
type UserFilter struct {
	Status string
	Role   string
	Search string
}

// StationFilter for filtering stations
type StationFilter struct {
	Status string
	Vendor string
	Search string
}

// TransactionFilter for filtering transactions
type TransactionFilter struct {
	Status        string
	UserID        string
	ChargePointID string
	StartDate     time.Time
	EndDate       time.Time
}

// UserDetails provides detailed user information
type UserDetails struct {
	User              *domain.User          `json:"user"`
	Wallet            *domain.Wallet        `json:"wallet,omitempty"`
	TotalTransactions int                   `json:"total_transactions"`
	TotalSpent        float64               `json:"total_spent"`
	TotalEnergyKWh    float64               `json:"total_energy_kwh"`
	LastActivity      *time.Time            `json:"last_activity,omitempty"`
	RecentTransactions []domain.Transaction `json:"recent_transactions,omitempty"`
}

// StationDetails provides detailed station information
type StationDetails struct {
	Station            *domain.ChargePoint   `json:"station"`
	Connectors         []domain.Connector    `json:"connectors"`
	TodayTransactions  int                   `json:"today_transactions"`
	TodayRevenue       float64               `json:"today_revenue"`
	TodayEnergyKWh     float64               `json:"today_energy_kwh"`
	Uptime             float64               `json:"uptime_percent"`
	LastHeartbeat      *time.Time            `json:"last_heartbeat,omitempty"`
	ActiveTransaction  *domain.Transaction   `json:"active_transaction,omitempty"`
	RecentTransactions []domain.Transaction  `json:"recent_transactions,omitempty"`
}

// TransactionDetails provides detailed transaction information
type TransactionDetails struct {
	Transaction *domain.Transaction `json:"transaction"`
	User        *domain.User        `json:"user,omitempty"`
	Station     *domain.ChargePoint `json:"station,omitempty"`
	Payment     *domain.Payment     `json:"payment,omitempty"`
	MeterValues []MeterValue        `json:"meter_values,omitempty"`
}

// MeterValue represents a meter reading
type MeterValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Context   string    `json:"context"`
}

// Alert represents a system alert
type Alert struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Title        string    `json:"title"`
	Message      string    `json:"message"`
	Source       string    `json:"source"`
	SourceID     string    `json:"source_id,omitempty"`
	Acknowledged bool      `json:"acknowledged"`
	CreatedAt    time.Time `json:"created_at"`
}
