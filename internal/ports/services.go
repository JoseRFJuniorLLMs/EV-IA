package ports

import (
	"context"
	"time"

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

// --- V2G (Vehicle-to-Grid) Services ---

// V2GService handles Vehicle-to-Grid operations
type V2GService interface {
	// StartDischarge initiates V2G discharge from vehicle to grid
	StartDischarge(ctx context.Context, req *V2GDischargeRequest) (*domain.V2GSession, error)

	// StopDischarge stops an active V2G discharge session
	StopDischarge(ctx context.Context, sessionID string) error

	// GetActiveSession returns the active V2G session for a charge point
	GetActiveSession(ctx context.Context, chargePointID string) (*domain.V2GSession, error)

	// GetSession returns a V2G session by ID
	GetSession(ctx context.Context, sessionID string) (*domain.V2GSession, error)

	// CalculateCompensation calculates user compensation for V2G session
	CalculateCompensation(ctx context.Context, session *domain.V2GSession) (*domain.V2GCompensation, error)

	// CheckV2GCapability checks if vehicle at charge point supports V2G
	CheckV2GCapability(ctx context.Context, chargePointID string) (*domain.V2GCapability, error)

	// SetUserPreferences sets V2G preferences for a user
	SetUserPreferences(ctx context.Context, userID string, prefs *domain.V2GPreferences) error

	// GetUserPreferences gets V2G preferences for a user
	GetUserPreferences(ctx context.Context, userID string) (*domain.V2GPreferences, error)

	// OptimizeV2G automatically optimizes V2G based on preferences and grid prices
	OptimizeV2G(ctx context.Context, chargePointID string, userID string) error

	// GetUserStats returns V2G statistics for a user
	GetUserStats(ctx context.Context, userID string, startDate, endDate time.Time) (*domain.V2GStats, error)
}

// V2GDischargeRequest represents a request to start V2G discharge
type V2GDischargeRequest struct {
	ChargePointID string     `json:"charge_point_id"`
	ConnectorID   int        `json:"connector_id"`
	UserID        string     `json:"user_id"`
	MaxPowerKW    float64    `json:"max_power_kw"`
	MaxEnergyKWh  float64    `json:"max_energy_kwh"`
	MinBatterySOC int        `json:"min_battery_soc"`
	EndTime       *time.Time `json:"end_time,omitempty"`
}

// GridPriceService handles grid electricity pricing
type GridPriceService interface {
	// GetCurrentPrice returns the current grid price in R$/kWh
	GetCurrentPrice(ctx context.Context) (float64, error)

	// GetPriceForecast returns price forecast for the next N hours
	GetPriceForecast(ctx context.Context, hours int) ([]domain.GridPricePoint, error)

	// IsPeakHour checks if current time is during peak hours
	IsPeakHour(ctx context.Context) (bool, error)

	// CalculateV2GCompensation calculates compensation for V2G discharge
	CalculateV2GCompensation(ctx context.Context, energyKWh float64, startTime, endTime time.Time) (float64, error)
}

// V2GRepository handles V2G data persistence
type V2GRepository interface {
	// Session operations
	CreateSession(ctx context.Context, session *domain.V2GSession) error
	UpdateSession(ctx context.Context, session *domain.V2GSession) error
	GetSession(ctx context.Context, sessionID string) (*domain.V2GSession, error)
	GetSessionsByChargePoint(ctx context.Context, chargePointID string, limit int) ([]domain.V2GSession, error)
	GetSessionsByUser(ctx context.Context, userID string, limit int) ([]domain.V2GSession, error)

	// Preferences operations
	SavePreferences(ctx context.Context, prefs *domain.V2GPreferences) error
	GetPreferences(ctx context.Context, userID string) (*domain.V2GPreferences, error)

	// Statistics
	GetUserStats(ctx context.Context, userID string, startDate, endDate time.Time) (*domain.V2GStats, error)
	GetChargePointStats(ctx context.Context, chargePointID string, startDate, endDate time.Time) (*domain.V2GStats, error)
}

// --- OCPP Command Service ---

// OCPPCommandService provides OCPP commands from CSMS to charge points
type OCPPCommandService interface {
	// RemoteStartTransaction requests charge point to start a transaction
	RemoteStartTransaction(ctx context.Context, chargePointID, idToken string, evseID *int) error

	// RemoteStopTransaction requests charge point to stop a transaction
	RemoteStopTransaction(ctx context.Context, chargePointID, transactionID string) error

	// Reset requests charge point to reset
	Reset(ctx context.Context, chargePointID string, resetType string, evseID *int) error

	// TriggerMessage requests charge point to send a specific message
	TriggerMessage(ctx context.Context, chargePointID, requestedMessage string, evseID *int) error

	// SetChargingProfile sets a charging profile on an EVSE
	SetChargingProfile(ctx context.Context, chargePointID string, evseID int, profile interface{}) error

	// ClearChargingProfile clears charging profile(s) from charge point
	ClearChargingProfile(ctx context.Context, chargePointID string, profileID *int, evseID *int) error

	// UpdateFirmware requests charge point to update firmware
	UpdateFirmware(ctx context.Context, chargePointID, firmwareURL, retrieveDateTime string, installDateTime *time.Time, retries, retryInterval *int) error

	// UpdateFirmwareSigned requests signed firmware update
	UpdateFirmwareSigned(ctx context.Context, chargePointID, firmwareURL, retrieveDateTime, signingCert, signature string, retries, retryInterval *int) error

	// UnlockConnector requests to unlock a connector
	UnlockConnector(ctx context.Context, chargePointID string, evseID, connectorID int) error

	// ChangeAvailability changes charge point/EVSE availability
	ChangeAvailability(ctx context.Context, chargePointID string, operationalStatus string, evseID *int) error

	// GetVariables retrieves variable values from charge point
	GetVariables(ctx context.Context, chargePointID string, variables []GetVariableRequest) ([]GetVariableResponse, error)

	// SetVariables sets variable values on charge point
	SetVariables(ctx context.Context, chargePointID string, variables []SetVariableRequest) error

	// GetLog requests diagnostic logs from charge point
	GetLog(ctx context.Context, chargePointID, logType, uploadURL string) error

	// V2G specific commands
	SetV2GChargingProfile(ctx context.Context, chargePointID string, evseID int, dischargePowerKW float64, durationSeconds int) error
	ClearV2GChargingProfile(ctx context.Context, chargePointID string, evseID int) error
	GetV2GCapability(ctx context.Context, chargePointID string) (*domain.V2GCapability, error)

	// Connection status
	IsConnected(chargePointID string) bool
	GetConnectedClients() []string
}

// GetVariableRequest for OCPP GetVariables
type GetVariableRequest struct {
	ComponentName string
	VariableName  string
	Instance      string
}

// GetVariableResponse for OCPP GetVariables response
type GetVariableResponse struct {
	ComponentName string
	VariableName  string
	Value         string
	Status        string
}

// SetVariableRequest for OCPP SetVariables
type SetVariableRequest struct {
	ComponentName string
	VariableName  string
	Value         string
}

// --- Firmware Service ---

// FirmwareService handles firmware update operations
type FirmwareService interface {
	// UpdateFirmware initiates a firmware update on a charge point
	UpdateFirmware(ctx context.Context, req *FirmwareUpdateRequest) (*FirmwareUpdateStatus, error)

	// GetFirmwareStatus returns current firmware update status
	GetFirmwareStatus(ctx context.Context, chargePointID string) (*FirmwareUpdateStatus, error)

	// CancelFirmwareUpdate attempts to cancel a firmware update
	CancelFirmwareUpdate(ctx context.Context, chargePointID string) error

	// HandleStatusNotification processes firmware status notifications
	HandleStatusNotification(chargePointID, status string, requestID *int) error
}

// FirmwareUpdateRequest for initiating firmware updates
type FirmwareUpdateRequest struct {
	ChargePointID      string     `json:"charge_point_id"`
	FirmwareURL        string     `json:"firmware_url"`
	Version            string     `json:"version"`
	RetrieveDateTime   *time.Time `json:"retrieve_datetime,omitempty"`
	InstallDateTime    *time.Time `json:"install_datetime,omitempty"`
	Retries            *int       `json:"retries,omitempty"`
	RetryInterval      *int       `json:"retry_interval,omitempty"`
	SigningCertificate string     `json:"signing_certificate,omitempty"`
	Signature          string     `json:"signature,omitempty"`
}

// FirmwareUpdateStatus represents firmware update status
type FirmwareUpdateStatus struct {
	ID            string    `json:"id"`
	ChargePointID string    `json:"charge_point_id"`
	Version       string    `json:"version"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// --- Message Queue Interface ---

// MessageQueue interface for publishing events
type MessageQueue interface {
	Publish(topic string, message interface{}) error
	Subscribe(topic string, handler func(message []byte)) error
	Close() error
}

// --- ISO 15118 (Plug & Charge) Services ---

// ISO15118Service handles ISO 15118 Plug & Charge operations
type ISO15118Service interface {
	// AuthenticateVehicle authenticates a vehicle using its ISO 15118 certificate
	AuthenticateVehicle(ctx context.Context, certChain []byte) (*domain.ISO15118VehicleIdentity, error)

	// ValidateCertificate validates an ISO 15118 certificate chain
	ValidateCertificate(ctx context.Context, certPEM []byte) error

	// GetChargingContract retrieves the charging contract for a vehicle
	GetChargingContract(ctx context.Context, emaid string) (*domain.ChargingContract, error)

	// RevokeCertificate revokes an ISO 15118 certificate
	RevokeCertificate(ctx context.Context, emaid, reason string) error

	// GetCertificateStatus gets the status of a certificate
	GetCertificateStatus(ctx context.Context, emaid string) (*ISO15118CertificateStatus, error)
}

// ISO15118CertificateStatus represents the status of an ISO 15118 certificate
type ISO15118CertificateStatus struct {
	EMAID            string     `json:"emaid"`
	ContractID       string     `json:"contract_id"`
	Valid            bool       `json:"valid"`
	Expired          bool       `json:"expired"`
	Revoked          bool       `json:"revoked"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason string     `json:"revocation_reason,omitempty"`
	ValidFrom        time.Time  `json:"valid_from"`
	ValidTo          time.Time  `json:"valid_to"`
	DaysUntilExpiry  int        `json:"days_until_expiry"`
	V2GCapable       bool       `json:"v2g_capable"`
}

// ISO15118Repository handles ISO 15118 certificate persistence
type ISO15118Repository interface {
	// StoreCertificate stores a new certificate
	StoreCertificate(ctx context.Context, cert interface{}) error

	// GetCertificateByEMAID retrieves a certificate by EMAID
	GetCertificateByEMAID(ctx context.Context, emaid string) (interface{}, error)

	// GetCertificateByContractID retrieves a certificate by contract ID
	GetCertificateByContractID(ctx context.Context, contractID string) (interface{}, error)

	// GetCertificateByVIN retrieves certificates by vehicle VIN
	GetCertificateByVIN(ctx context.Context, vin string) ([]interface{}, error)

	// UpdateCertificate updates an existing certificate
	UpdateCertificate(ctx context.Context, cert interface{}) error

	// GetExpiringCertificates retrieves certificates expiring within N days
	GetExpiringCertificates(ctx context.Context, daysUntilExpiry int) ([]interface{}, error)

	// GetV2GCapableCertificates retrieves all V2G-capable certificates
	GetV2GCapableCertificates(ctx context.Context) ([]interface{}, error)
}
