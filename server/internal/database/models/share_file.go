package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShareFile struct {
	ID          string `gorm:"primaryKey" json:"id"`
	ShareID     string `gorm:"index;not null" json:"shareId"`
	Name        string `gorm:"not null" json:"name"`
	Size        int64  `gorm:"not null" json:"size"`
	ContentType string `json:"contentType"`

	ByOwner bool `gorm:"not null;default:false" json:"byOwner"`

	// UploaderLabel is optional, self-declared and unverified. It is displayed
	// only to the Owner and must never be rendered as markup.
	UploaderLabel string `json:"uploaderLabel"`

	// UploadSessionID groups a visitor's single drop, so per-file outcomes can be
	// reported together and exactly one notification is sent per batch.
	UploadSessionID string `gorm:"index" json:"-"`

	UploadedAt time.Time `json:"uploadedAt"`
}

func (f *ShareFile) BeforeCreate(*gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	if f.UploadedAt.IsZero() {
		f.UploadedAt = time.Now()
	}
	return nil
}

func (ShareFile) TableName() string { return "share_file" }

// PendingUpload records an upload in flight. It does three jobs that would
// otherwise each need their own mechanism: it reserves quota for bytes not yet
// on disk, it gives the abandoned-upload sweep a set to enumerate instead of
// scanning the upload directory, and it is the trusted server-side binding from
// an upload to its target Share — never taken from client-supplied tus metadata.
//
// ShareID is empty for an upload that has no Share yet: an Owner dropping files
// to send creates one when the first file lands, so a drop where every file
// fails leaves nothing behind. NewShareTitle carries the title to give it.
type PendingUpload struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	OwnerID         string    `gorm:"index;not null" json:"ownerId"`
	ShareID         string    `gorm:"index;not null" json:"shareId"`
	NewShareTitle   string    `json:"newShareTitle"`
	DeclaredSize    int64     `gorm:"not null" json:"declaredSize"`
	UploadSessionID string    `gorm:"index" json:"uploadSessionId"`
	ByOwner         bool      `gorm:"not null;default:false" json:"byOwner"`
	UploaderLabel   string    `json:"uploaderLabel"`
	Name            string    `json:"name"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (PendingUpload) TableName() string { return "pending_upload" }
