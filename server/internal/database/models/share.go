package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShareMode is exclusive and immutable once a Share exists.
//
// A hybrid was considered and rejected: allowing uploads into a Send Share
// forces a third trust promise, because a link-holder who may read the Owner's
// files either can or cannot read what other link-holders contributed.
// Immutability follows — files are contributed under a promise about who may read
// them, and flipping mode would retroactively break it.
type ShareMode string

const (
	ShareModeSend      ShareMode = "send"
	ShareModeUploadBox ShareMode = "upload_box"
)

func (m ShareMode) Valid() bool {
	return m == ShareModeSend || m == ShareModeUploadBox
}

type Share struct {
	ID      string    `gorm:"primaryKey" json:"id"`
	OwnerID string    `gorm:"index;not null" json:"ownerId"`
	Title   string    `gorm:"not null" json:"title"`
	Mode    ShareMode `gorm:"not null" json:"mode"`

	// AccessToken is the public capability, and the only one of these that ever
	// appears in a URL. Rotating it revokes every distributed copy of the link
	// while the Share, its files and its stats survive.
	AccessToken string `gorm:"uniqueIndex;not null" json:"-"`

	// Password is stored in the clear, deliberately, so an owner can look up what
	// they set on a share. Anyone with database or backup access can therefore read
	// it. User account passwords are bcrypt-hashed and are not affected by this.
	Password string `json:"-"`

	// UnlockToken is regenerated whenever the password changes, which is what
	// makes a password change invalidate every outstanding unlock cookie.
	UnlockToken string `json:"-"`

	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	MaxDownloads *int       `json:"maxDownloads,omitempty"`
	Downloads    int        `gorm:"not null;default:0" json:"downloads"`

	MaxFiles *int   `json:"maxFiles,omitempty"`
	MaxBytes *int64 `json:"maxBytes,omitempty"`

	NotifyURL         string     `json:"notifyUrl"`
	NotifyEnabled     bool       `gorm:"not null;default:false" json:"notifyEnabled"`
	NotifyLastError   string     `json:"notifyLastError"`
	NotifyLastErrorAt *time.Time `json:"notifyLastErrorAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Files []ShareFile `gorm:"foreignKey:ShareID" json:"files"`
}

func (s *Share) BeforeCreate(*gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

func (Share) TableName() string { return "share" }

func (s *Share) HasPassword() bool { return s.Password != "" }

func (s *Share) Expired() bool {
	return s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt)
}

func (s *Share) DownloadsExhausted() bool {
	return s.MaxDownloads != nil && s.Downloads >= *s.MaxDownloads
}

// PublicAccessClosed reports whether the link has stopped serving. It says
// nothing about the Owner's access, which expiry never revokes.
func (s *Share) PublicAccessClosed() bool {
	return s.Expired() || s.DownloadsExhausted()
}

// RangeRequestsAllowed is false once a download cap is set. Without this, two
// partial Range requests could reassemble a file locally and never increment the
// counter, so a cap of 1 would bound nothing. The cost is that a capped Share
// loses resumable downloads.
func (s *Share) RangeRequestsAllowed() bool {
	return s.MaxDownloads == nil
}

// SweepDueAt is when this Share's blobs may be deleted: its expiry plus the
// grace period during which the Owner can still extend and reopen it.
func (s *Share) SweepDueAt(grace time.Duration) *time.Time {
	if s.ExpiresAt == nil {
		return nil
	}
	due := s.ExpiresAt.Add(grace)
	return &due
}
