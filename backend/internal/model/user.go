package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	Nickname        *string    `json:"nickname" db:"nickname"`
	Avatar          *string    `json:"avatar" db:"avatar"`
	Role            string     `json:"role" db:"role"`
	Status          string     `json:"status" db:"status"`
	AccountNumber   *string    `json:"account_number" db:"account_number"`
	EmailVerifiedAt *time.Time `json:"email_verified_at" db:"email_verified_at"`
	LastLoginAt     *time.Time `json:"last_login_at" db:"last_login_at"`
	DeletedAt       *time.Time `json:"deleted_at" db:"deleted_at"`
	TokenVersion    int        `json:"token_version" db:"token_version"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}
