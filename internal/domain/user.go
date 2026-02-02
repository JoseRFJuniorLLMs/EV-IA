package domain

import (
	"time"
)

type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleOperator UserRole = "operator"
	UserRoleUser     UserRole = "user"
)

type User struct {
	ID                string    `json:"id" gorm:"primaryKey"`
	Name              string    `json:"name"`
	Email             string    `json:"email" gorm:"uniqueIndex"`
	Phone             string    `json:"phone,omitempty" gorm:"index"`
	PhoneVerified     bool      `json:"phone_verified"`
	Password          string    `json:"-"` // Hashed password
	Role              UserRole  `json:"role"`
	Status            string    `json:"status"` // Active, Inactive, Blocked
	NotifyByEmail     bool      `json:"notify_by_email" gorm:"default:true"`
	NotifyByWhatsApp  bool      `json:"notify_by_whatsapp" gorm:"default:false"`
	NotifyBySMS       bool      `json:"notify_by_sms" gorm:"default:false"`
	PreferredLanguage string    `json:"preferred_language" gorm:"default:pt-BR"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
