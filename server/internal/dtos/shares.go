package dtos

import "time"

// CreateShareDto creates a Share. Mode is fixed here and can never change: files
// are contributed under a promise about who may read them.
//
// Throughout these DTOs, a nil pointer means "leave unchanged" and zero means
// "no limit" — matching the convention the env vars already use.
type CreateShareDto struct {
	Title         string  `json:"title"`
	Mode          string  `json:"mode" validate:"required,oneof=send upload_box"`
	ExpiresInDays *int    `json:"expiresInDays,omitempty"`
	MaxDownloads  *int    `json:"maxDownloads,omitempty"`
	MaxFiles      *int    `json:"maxFiles,omitempty"`
	MaxBytes      *int64  `json:"maxBytes,omitempty"`
	Password      *string `json:"password,omitempty"`
	NotifyURL     *string `json:"notifyUrl,omitempty"`
}

type UpdateShareDto struct {
	Title         *string `json:"title,omitempty"`
	ExpiresInDays *int    `json:"expiresInDays,omitempty"`
	MaxDownloads  *int    `json:"maxDownloads,omitempty"`
	MaxFiles      *int    `json:"maxFiles,omitempty"`
	MaxBytes      *int64  `json:"maxBytes,omitempty"`
	Password      *string `json:"password,omitempty"`
	NotifyURL     *string `json:"notifyUrl,omitempty"`
	NotifyEnabled *bool   `json:"notifyEnabled,omitempty"`
}

type ShareFileDto struct {
	ID            string    `json:"id" validate:"required"`
	Name          string    `json:"name" validate:"required"`
	Size          int64     `json:"size" validate:"required"`
	ContentType   string    `json:"contentType" validate:"required"`
	ByOwner       bool      `json:"byOwner" validate:"required"`
	UploaderLabel string    `json:"uploaderLabel" validate:"required"`
	UploadedAt    time.Time `json:"uploadedAt" validate:"required"`
	DownloadPath  string    `json:"downloadPath" validate:"required"`
}

// ShareDto is the Owner's view. It carries the public path plus an absolute
// publicUrl when PUBLIC_BASE_URL is configured; the browser composes the
// private URL from its own origin, which it knows more reliably than the server
// does behind a proxy.
type ShareDto struct {
	ID    string `json:"id" validate:"required"`
	Title string `json:"title" validate:"required"`
	Mode  string `json:"mode" validate:"required"`

	Path      string `json:"path" validate:"required"`
	PublicURL string `json:"publicUrl" validate:"required"`

	HasPassword bool `json:"hasPassword" validate:"required"`
	// Password is returned in the clear so an owner can look up what they set.
	// ShareDto is only ever served to the share's owner; PublicShareDto, which
	// strangers receive, deliberately has no password field at all.
	Password  string     `json:"password" validate:"required"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Expired   bool       `json:"expired" validate:"required"`

	MaxDownloads *int `json:"maxDownloads,omitempty"`
	Downloads    int  `json:"downloads" validate:"required"`

	MaxFiles *int   `json:"maxFiles,omitempty"`
	MaxBytes *int64 `json:"maxBytes,omitempty"`

	NotifyURL         string     `json:"notifyUrl" validate:"required"`
	NotifyEnabled     bool       `json:"notifyEnabled" validate:"required"`
	NotifyLastError   string     `json:"notifyLastError" validate:"required"`
	NotifyLastErrorAt *time.Time `json:"notifyLastErrorAt,omitempty"`

	TotalBytes int64          `json:"totalBytes" validate:"required"`
	Files      []ShareFileDto `json:"files" validate:"required"`

	CreatedAt time.Time `json:"createdAt" validate:"required"`
}

// PublicShareDto is what a stranger sees. For an Upload Box it deliberately
// carries no counts at all: "17 of 20 remaining" would reveal that three other
// people have already uploaded, which for sealed submissions is real signal.
type PublicShareDto struct {
	Title string `json:"title" validate:"required"`
	Mode  string `json:"mode" validate:"required"`

	// State is one of: open, locked, expired, full.
	State string `json:"state" validate:"required"`

	RequiresPassword bool `json:"requiresPassword" validate:"required"`
	Unlocked         bool `json:"unlocked" validate:"required"`

	// Files and TotalBytes are populated only for an unlocked Send Share.
	Files      []PublicFileDto `json:"files" validate:"required"`
	TotalBytes int64           `json:"totalBytes" validate:"required"`
	ZipPath    string          `json:"zipPath" validate:"required"`
}

type PublicFileDto struct {
	Name         string `json:"name" validate:"required"`
	Size         int64  `json:"size" validate:"required"`
	DownloadPath string `json:"downloadPath" validate:"required"`
}

type UnlockDto struct {
	Password string `json:"password" validate:"required"`
}

// UploadOutcomeDto reports one file's fate so partial acceptance is legible to
// the uploader rather than mysterious.
type UploadOutcomeDto struct {
	Name     string `json:"name" validate:"required"`
	Size     int64  `json:"size" validate:"required"`
	Accepted bool   `json:"accepted" validate:"required"`
	Reason   string `json:"reason" validate:"required"`
}

type FinalizeUploadDto struct {
	UploadSessionID string `json:"uploadSessionId" validate:"required"`
}

type FinalizeResultDto struct {
	Outcomes []UploadOutcomeDto `json:"outcomes" validate:"required"`
}

type UsageDto struct {
	UsedBytes  int64 `json:"usedBytes" validate:"required"`
	QuotaBytes int64 `json:"quotaBytes" validate:"required"`
	ShareCount int   `json:"shareCount" validate:"required"`
}

// StorageRowDto is the admin view: bytes and counts only. Never titles,
// filenames or uploader labels, so Upload Box privacy holds against admins.
type StorageRowDto struct {
	UserID     string `json:"userId" validate:"required"`
	Username   string `json:"username" validate:"required"`
	UsedBytes  int64  `json:"usedBytes" validate:"required"`
	QuotaBytes int64  `json:"quotaBytes" validate:"required"`
	ShareCount int    `json:"shareCount" validate:"required"`
}

type StorageOverviewDto struct {
	Users     []StorageRowDto `json:"users" validate:"required"`
	FreeBytes int64           `json:"freeBytes" validate:"required"`
}

// CliUploadResultDto is what the curl path prints. It carries the recipient link
// and the Owner's management link; both prefer PUBLIC_BASE_URL, since a shell has
// no window.location for the server to defer to.
type CliUploadResultDto struct {
	ShareID   string     `json:"shareId"`
	FileID    string     `json:"fileId"`
	URL       string     `json:"url"`
	ManageURL string     `json:"manageUrl"`
	Size      int64      `json:"size"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}
