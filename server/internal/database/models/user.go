package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User is an account. IsAdmin grants server administration only — never access
// to another account's Shares. Every Share query filters on owner_id with no
// branch on IsAdmin, which is what makes Upload Box privacy hold against admins.
type User struct {
	ID           string `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"not null" json:"-"`
	IsAdmin      bool   `gorm:"not null;default:false" json:"isAdmin"`

	// QuotaBytes nil means inherit the server default; 0 means unlimited.
	QuotaBytes *int64 `json:"quotaBytes,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

func (u *User) BeforeCreate(*gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}

func (User) TableName() string { return "user" }
