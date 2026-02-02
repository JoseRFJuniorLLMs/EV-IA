package domain

import (
	"time"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// PaymentMethod represents the payment method type
type PaymentMethod string

const (
	PaymentMethodCreditCard PaymentMethod = "credit_card"
	PaymentMethodDebitCard  PaymentMethod = "debit_card"
	PaymentMethodPix        PaymentMethod = "pix"
	PaymentMethodBoleto     PaymentMethod = "boleto"
	PaymentMethodWallet     PaymentMethod = "wallet"
)

// PaymentProvider represents the payment provider
type PaymentProvider string

const (
	PaymentProviderStripe    PaymentProvider = "stripe"
	PaymentProviderPagSeguro PaymentProvider = "pagseguro"
)

// Payment represents a payment transaction
type Payment struct {
	ID              string          `json:"id" gorm:"primaryKey"`
	UserID          string          `json:"user_id" gorm:"index"`
	TransactionID   string          `json:"transaction_id,omitempty" gorm:"index"`
	Provider        PaymentProvider `json:"provider"`
	ProviderID      string          `json:"provider_id"` // External payment ID
	Method          PaymentMethod   `json:"method"`
	Status          PaymentStatus   `json:"status"`
	Amount          float64         `json:"amount"`
	Currency        string          `json:"currency"`
	Description     string          `json:"description,omitempty"`
	FailureReason   string          `json:"failure_reason,omitempty"`
	Metadata        JSONMap         `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
}

// PaymentCard represents a stored payment card
type PaymentCard struct {
	ID           string          `json:"id" gorm:"primaryKey"`
	UserID       string          `json:"user_id" gorm:"index"`
	Provider     PaymentProvider `json:"provider"`
	ProviderID   string          `json:"provider_id"` // Card token from provider
	Brand        string          `json:"brand"`       // visa, mastercard, etc
	Last4        string          `json:"last4"`
	ExpMonth     int             `json:"exp_month"`
	ExpYear      int             `json:"exp_year"`
	HolderName   string          `json:"holder_name"`
	IsDefault    bool            `json:"is_default"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Wallet represents a user's wallet/balance
type Wallet struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"uniqueIndex"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WalletTransaction represents a wallet transaction
type WalletTransaction struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WalletID    string    `json:"wallet_id" gorm:"index"`
	UserID      string    `json:"user_id" gorm:"index"`
	Type        string    `json:"type"` // credit, debit
	Amount      float64   `json:"amount"`
	Balance     float64   `json:"balance"` // Balance after transaction
	Description string    `json:"description"`
	ReferenceID string    `json:"reference_id,omitempty"` // Payment or Transaction ID
	CreatedAt   time.Time `json:"created_at"`
}

// Refund represents a payment refund
type Refund struct {
	ID         string        `json:"id" gorm:"primaryKey"`
	PaymentID  string        `json:"payment_id" gorm:"index"`
	ProviderID string        `json:"provider_id"`
	Amount     float64       `json:"amount"`
	Status     PaymentStatus `json:"status"`
	Reason     string        `json:"reason,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}

// JSONMap is a helper type for JSONB columns
type JSONMap map[string]interface{}

// PaymentIntent represents a payment intent for client-side confirmation
type PaymentIntent struct {
	ID           string  `json:"id"`
	ClientSecret string  `json:"client_secret"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Status       string  `json:"status"`
}

// PixPayment represents PIX payment details
type PixPayment struct {
	QRCode      string    `json:"qr_code"`
	QRCodeImage string    `json:"qr_code_image"` // Base64 encoded
	CopyPaste   string    `json:"copy_paste"`    // PIX copia e cola
	ExpiresAt   time.Time `json:"expires_at"`
}

// BoletoPayment represents Boleto payment details
type BoletoPayment struct {
	Barcode     string    `json:"barcode"`
	BoletoURL   string    `json:"boleto_url"`
	DigitableLine string  `json:"digitable_line"`
	ExpiresAt   time.Time `json:"expires_at"`
}
