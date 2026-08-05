package services

import (
	"crypto/subtle"
	"strings"
	"sync"
	"time"

	"app/internal/config"
	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/repositories"
	"app/internal/storage"
	"app/internal/tokens"
)

type SharesService struct {
	shares  *repositories.SharesRepository
	users   *repositories.UsersRepository
	storage *storage.Storage
	notify  *NotifyService

	throttle *guessThrottle
}

func NewSharesService(
	shares *repositories.SharesRepository,
	users *repositories.UsersRepository,
	store *storage.Storage,
	notify *NotifyService,
) *SharesService {
	return &SharesService{
		shares:   shares,
		users:    users,
		storage:  store,
		notify:   notify,
		throttle: newGuessThrottle(),
	}
}

// guessThrottle slows password guessing on a Share.
//
// It delays rather than locks out, because a lockout would let one attacker deny
// access to every legitimate recipient of that link. It is keyed per Share and
// not per IP: per-IP is trivially bypassed, and dropla deliberately stores no IP
// addresses. Counters live in memory, so a restart clears them — which costs an
// attacker a restart they cannot trigger and avoids a database write per guess.
type guessThrottle struct {
	mu       sync.Mutex
	failures map[string]int
}

func newGuessThrottle() *guessThrottle {
	return &guessThrottle{failures: map[string]int{}}
}

func (t *guessThrottle) delay(shareID string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	n := t.failures[shareID]
	if n < 3 {
		return 0
	}

	d := time.Duration(n-2) * 250 * time.Millisecond
	if d > 3*time.Second {
		d = 3 * time.Second
	}
	return d
}

func (t *guessThrottle) fail(shareID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failures[shareID]++
}

func (t *guessThrottle) succeed(shareID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.failures, shareID)
}

func (s *SharesService) List(ownerID string) ([]models.Share, error) {
	return s.shares.ListForOwner(ownerID)
}

// Owned fetches a Share and enforces ownership. There is no admin bypass: the
// same rule applies to every account, which is what makes Upload Box privacy
// hold against admins too.
func (s *SharesService) Owned(ownerID, shareID string) (*models.Share, error) {
	share, err := s.shares.Get(shareID)
	if err != nil {
		return nil, err
	}
	if share == nil || share.OwnerID != ownerID {
		return nil, ErrShareNotFound
	}
	return share, nil
}

func (s *SharesService) Create(ownerID string, dto dtos.CreateShareDto) (*models.Share, error) {
	mode := models.ShareMode(dto.Mode)
	if !mode.Valid() {
		return nil, ErrNotFound
	}

	title := strings.TrimSpace(dto.Title)
	if title == "" {
		title = "Untitled"
	}

	share := &models.Share{
		OwnerID:     ownerID,
		Title:       title,
		Mode:        mode,
		AccessToken: tokens.Share(),
	}

	if dto.ExpiresInDays != nil && *dto.ExpiresInDays > 0 {
		at := time.Now().Add(time.Duration(*dto.ExpiresInDays) * 24 * time.Hour)
		share.ExpiresAt = &at
	}
	share.MaxDownloads = positiveOrNil(dto.MaxDownloads)
	share.MaxFiles = positiveOrNil(dto.MaxFiles)
	share.MaxBytes = positiveOrNil64(dto.MaxBytes)

	if dto.NotifyURL != nil {
		if url := strings.TrimSpace(*dto.NotifyURL); url != "" {
			share.NotifyURL = url
			share.NotifyEnabled = true
		}
	}

	if dto.Password != nil && *dto.Password != "" {
		share.Password = *dto.Password
		share.UnlockToken = tokens.Unlock()
	}

	if err := s.shares.Create(share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *SharesService) Update(ownerID, shareID string, dto dtos.UpdateShareDto) (*models.Share, error) {
	share, err := s.Owned(ownerID, shareID)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}

	if dto.Title != nil {
		if title := strings.TrimSpace(*dto.Title); title != "" {
			fields["title"] = title
		}
	}

	if dto.ExpiresInDays != nil {
		if *dto.ExpiresInDays <= 0 {
			fields["expires_at"] = nil
		} else {
			fields["expires_at"] = time.Now().Add(time.Duration(*dto.ExpiresInDays) * 24 * time.Hour)
		}
	}

	if dto.MaxDownloads != nil {
		fields["max_downloads"] = positiveOrNil(dto.MaxDownloads)
	}
	if dto.MaxFiles != nil {
		fields["max_files"] = positiveOrNil(dto.MaxFiles)
	}
	if dto.MaxBytes != nil {
		fields["max_bytes"] = positiveOrNil64(dto.MaxBytes)
	}
	if dto.NotifyURL != nil {
		fields["notify_url"] = strings.TrimSpace(*dto.NotifyURL)
		fields["notify_last_error"] = ""
		fields["notify_last_error_at"] = nil
	}
	if dto.NotifyEnabled != nil {
		fields["notify_enabled"] = *dto.NotifyEnabled
	}

	// Setting or clearing a password regenerates the unlock token, which is what
	// makes the change revoke every outstanding unlock cookie.
	if dto.Password != nil {
		if *dto.Password == "" {
			fields["password"] = ""
			fields["unlock_token"] = ""
		} else {
			fields["password"] = *dto.Password
			fields["unlock_token"] = tokens.Unlock()
		}
	}

	if len(fields) > 0 {
		if err := s.shares.Update(share.ID, fields); err != nil {
			return nil, err
		}
	}
	return s.shares.Get(share.ID)
}

// RotateLink revokes every distributed copy of the URL while the Share, its
// files and its download history survive.
func (s *SharesService) RotateLink(ownerID, shareID string) (*models.Share, error) {
	share, err := s.Owned(ownerID, shareID)
	if err != nil {
		return nil, err
	}
	if err := s.shares.Update(share.ID, map[string]any{"access_token": tokens.Share()}); err != nil {
		return nil, err
	}
	return s.shares.Get(share.ID)
}

func (s *SharesService) Delete(ownerID, shareID string) error {
	share, err := s.Owned(ownerID, shareID)
	if err != nil {
		return err
	}
	if err := s.shares.Delete(share.ID); err != nil {
		return err
	}
	// An explicit deletion is immediate and gets no grace period: the Owner asked.
	return s.storage.RemoveShare(share.ID)
}

func (s *SharesService) DeleteFile(ownerID, shareID, fileID string) error {
	share, err := s.Owned(ownerID, shareID)
	if err != nil {
		return err
	}

	deleted, err := s.shares.DeleteFile(share.ID, fileID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return s.storage.Remove(share.ID, fileID)
}

// ResolvePublic looks a Share up by the capability in its URL.
func (s *SharesService) ResolvePublic(token string) (*models.Share, error) {
	share, err := s.shares.GetByAccessToken(token)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, ErrShareNotFound
	}
	return share, nil
}

// Unlocked reports whether a visitor's cookie matches the Share's current unlock
// token. A Share with no password is always unlocked.
func (s *SharesService) Unlocked(share *models.Share, cookieValue string) bool {
	if !share.HasPassword() {
		return true
	}
	return share.UnlockToken != "" && cookieValue == share.UnlockToken
}

// Unlock verifies a password and returns the value to store in the cookie.
func (s *SharesService) Unlock(share *models.Share, password string) (string, error) {
	if !share.HasPassword() {
		return "", ErrNotFound
	}

	if d := s.throttle.delay(share.ID); d > 0 {
		time.Sleep(d)
	}

	// Constant-time so a wrong guess cannot be narrowed down by response timing.
	// bcrypt's cost previously provided a brute-force floor; with the password
	// stored in the clear, the per-share throttle above is what remains.
	if subtle.ConstantTimeCompare([]byte(share.Password), []byte(password)) != 1 {
		s.throttle.fail(share.ID)
		return "", ErrWrongPassword
	}

	s.throttle.succeed(share.ID)
	return share.UnlockToken, nil
}

// PublicState is the single word a stranger is told about a Share. "full" and
// "the Owner is out of quota" deliberately collapse into the same value: the
// second is not the visitor's business.
func (s *SharesService) PublicState(share *models.Share, unlocked bool) (string, error) {
	if share.PublicAccessClosed() {
		return "expired", nil
	}
	if share.HasPassword() && !unlocked {
		return "locked", nil
	}
	if share.Mode == models.ShareModeUploadBox {
		accepting, err := s.uploadBoxAccepting(share)
		if err != nil {
			return "", err
		}
		if !accepting {
			return "full", nil
		}
	}
	return "open", nil
}

func (s *SharesService) uploadBoxAccepting(share *models.Share) (bool, error) {
	bytes, files, err := s.shares.ShareUsage(share.ID)
	if err != nil {
		return false, err
	}
	if share.MaxFiles != nil && files >= *share.MaxFiles {
		return false, nil
	}
	if share.MaxBytes != nil && bytes >= *share.MaxBytes {
		return false, nil
	}

	owner, err := s.users.GetByID(share.OwnerID)
	if err != nil || owner == nil {
		return false, err
	}
	quota := s.quotaFor(owner)
	if quota <= 0 {
		return true, nil
	}

	used, err := s.shares.OwnerUsage(share.OwnerID)
	if err != nil {
		return false, err
	}
	return used < quota, nil
}

func (s *SharesService) quotaFor(user *models.User) int64 {
	if user.QuotaBytes != nil {
		return *user.QuotaBytes
	}
	return config.DefaultUserQuotaBytes()
}

// CheckUploadAllowed enforces every numeric limit before a byte is transferred.
// declaredSize may be zero when the client cannot say (a piped stdin upload), in
// which case only the caps that do not depend on size are checked here and the
// running total is enforced while streaming.
func (s *SharesService) CheckUploadAllowed(share *models.Share, declaredSize int64) error {
	bytes, files, err := s.shares.ShareUsage(share.ID)
	if err != nil {
		return err
	}
	if share.MaxFiles != nil && files >= *share.MaxFiles {
		return ErrUploadBoxFull
	}
	if share.MaxBytes != nil && bytes+declaredSize > *share.MaxBytes {
		return ErrUploadBoxFull
	}
	return s.CheckOwnerUploadAllowed(share.OwnerID, declaredSize)
}

// CheckOwnerUploadAllowed enforces the limits that belong to the account rather
// than to one Share, which is all there is to check for an upload whose Share
// does not exist yet.
func (s *SharesService) CheckOwnerUploadAllowed(ownerID string, declaredSize int64) error {
	if max := config.MaxUploadBytes(); max > 0 && declaredSize > max {
		return ErrFileTooLarge
	}

	owner, err := s.users.GetByID(ownerID)
	if err != nil {
		return err
	}
	if owner == nil {
		return ErrShareNotFound
	}

	quota := s.quotaFor(owner)
	if quota <= 0 {
		return nil
	}

	// Usage already includes in-flight reservations, so two concurrent uploads
	// cannot both pass a check they would jointly breach.
	used, err := s.shares.OwnerUsage(ownerID)
	if err != nil {
		return err
	}
	if used+declaredSize > quota {
		return ErrQuotaExceeded
	}
	return nil
}

// RemainingQuota reports how many bytes the Owner may still store, or -1 when
// unlimited. Used to enforce a running total for uploads of unknown length.
func (s *SharesService) RemainingQuota(share *models.Share) (int64, error) {
	owner, err := s.users.GetByID(share.OwnerID)
	if err != nil || owner == nil {
		return 0, err
	}

	quota := s.quotaFor(owner)
	if quota <= 0 {
		return -1, nil
	}

	used, err := s.shares.OwnerUsage(share.OwnerID)
	if err != nil {
		return 0, err
	}
	if remaining := quota - used; remaining > 0 {
		return remaining, nil
	}
	return 0, nil
}

func (s *SharesService) Reserve(p *models.PendingUpload) error {
	p.ID = uuidString()
	return s.shares.CreatePendingUpload(p)
}

func (s *SharesService) ReserveWithID(p *models.PendingUpload) error {
	return s.shares.CreatePendingUpload(p)
}

func (s *SharesService) Pending(id string) (*models.PendingUpload, error) {
	return s.shares.GetPendingUpload(id)
}

func (s *SharesService) ReleasePending(id string) error {
	return s.shares.DeletePendingUpload(id)
}

// ShareForPending resolves the Share a finished upload belongs to, creating it
// for a drop that started without one so that a Share only ever exists once a
// file has actually arrived.
//
// Creating here is safe without locking because tus delivers completions on a
// single goroutine: two files of one drop can never both find the Share missing.
func (s *SharesService) ShareForPending(p *models.PendingUpload) (*models.Share, error) {
	if p.ShareID != "" {
		return s.shares.Get(p.ShareID)
	}

	share, err := s.shares.ShareForUploadSession(p.UploadSessionID)
	if err != nil || share != nil {
		return share, err
	}
	return s.Create(p.OwnerID, dtos.CreateShareDto{
		Title: p.NewShareTitle,
		Mode:  string(models.ShareModeSend),
	})
}

// ShareForUploadSession is how a client learns which Share its drop created.
func (s *SharesService) ShareForUploadSession(ownerID, sessionID string) (*models.Share, error) {
	share, err := s.shares.ShareForUploadSession(sessionID)
	if err != nil {
		return nil, err
	}
	if share == nil || share.OwnerID != ownerID {
		return nil, ErrShareNotFound
	}
	return share, nil
}

// Commit turns a finished upload into a file in the Share and releases its
// reservation.
func (s *SharesService) Commit(p *models.PendingUpload, shareID, name string, size int64, contentType string) (*models.ShareFile, error) {
	file := &models.ShareFile{
		ShareID:         shareID,
		Name:            name,
		Size:            size,
		ContentType:     contentType,
		ByOwner:         p.ByOwner,
		UploaderLabel:   p.UploaderLabel,
		UploadSessionID: p.UploadSessionID,
	}
	if err := s.shares.CreateFile(file); err != nil {
		return nil, err
	}
	if err := s.shares.DeletePendingUpload(p.ID); err != nil {
		return nil, err
	}
	return file, nil
}

func (s *SharesService) File(shareID, fileID string) (*models.ShareFile, error) {
	file, err := s.shares.GetFile(shareID, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrNotFound
	}
	return file, nil
}

func (s *SharesService) CountDownload(shareID string) error {
	return s.shares.IncrementDownloads(shareID)
}

// Usage is for display: bytes actually stored, excluding uploads in flight.
func (s *SharesService) Usage(ownerID string) (int64, error) {
	return s.shares.OwnerCommittedBytes(ownerID)
}

func (s *SharesService) ShareUsage(shareID string) (int64, int, error) {
	return s.shares.ShareUsage(shareID)
}

func (s *SharesService) FilesForSession(sessionID string) ([]models.ShareFile, error) {
	return s.shares.FilesForSession(sessionID)
}

// NotifyArrival sends exactly one notification per upload session.
func (s *SharesService) NotifyArrival(share *models.Share, fileCount int, label string) {
	s.notify.UploadBoxArrival(share, fileCount, label)
}

func positiveOrNil(v *int) *int {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}

func positiveOrNil64(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}
