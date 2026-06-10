package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"`
	Nickname      *string    `json:"nickname" db:"nickname"`
	Avatar        *string    `json:"avatar" db:"avatar"`
	Role          string     `json:"role" db:"role"`
	Status        string     `json:"status" db:"status"`
	AccountNumber *string    `json:"account_number" db:"account_number"`
	LastLoginAt   *time.Time `json:"last_login_at" db:"last_login_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}
