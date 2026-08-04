package models

import "time"

type Session struct {
	TokenHash string    `gorm:"primaryKey" json:"-"`
	UserID    string    `gorm:"index;not null" json:"userId"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
}

func (Session) TableName() string { return "session" }

// AccessToken authorises CLI uploads. Kept separate from the password so a
// leaked script credential can be revoked on its own — changing a password drops
// every session, which would otherwise be the only remedy.
type AccessToken struct {
	ID         string     `gorm:"primaryKey" json:"id"`
	UserID     string     `gorm:"index;not null" json:"-"`
	Name       string     `gorm:"not null" json:"name"`
	TokenHash  string     `gorm:"uniqueIndex;not null" json:"-"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

func (AccessToken) TableName() string { return "access_token" }
