package repositories

import (
	"time"

	"app/internal/database/models"

	"gorm.io/gorm"
)

type SharesRepository struct {
	db *gorm.DB
}

func NewSharesRepository(db *gorm.DB) *SharesRepository {
	return &SharesRepository{db: db}
}

func (r *SharesRepository) ListForOwner(ownerID string) ([]models.Share, error) {
	shares := []models.Share{}
	err := r.db.Preload("Files").Where("owner_id = ?", ownerID).
		Order("created_at desc").Find(&shares).Error
	return shares, err
}

// Get looks a Share up by identity. Callers acting for a user must still check
// ownership; there is no admin bypass anywhere in this layer.
func (r *SharesRepository) Get(id string) (*models.Share, error) {
	s := models.Share{}
	return notFoundAsNil(&s, r.db.Preload("Files").First(&s, "id = ?", id).Error)
}

// GetByAccessToken resolves the public capability that appears in /s/<token>.
func (r *SharesRepository) GetByAccessToken(token string) (*models.Share, error) {
	s := models.Share{}
	return notFoundAsNil(&s, r.db.Preload("Files").First(&s, "access_token = ?", token).Error)
}

func (r *SharesRepository) Create(s *models.Share) error {
	return r.db.Create(s).Error
}

// Update writes named columns so zero values — clearing an expiry, removing a
// password — persist where a struct update would skip them.
func (r *SharesRepository) Update(id string, fields map[string]any) error {
	return r.db.Model(&models.Share{}).Where("id = ?", id).Updates(fields).Error
}

func (r *SharesRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.ShareFile{}, "share_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.PendingUpload{}, "share_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Share{}, "id = ?", id).Error
	})
}

func (r *SharesRepository) IncrementDownloads(id string) error {
	return r.db.Model(&models.Share{}).Where("id = ?", id).
		UpdateColumn("downloads", gorm.Expr("downloads + 1")).Error
}

func (r *SharesRepository) CreateFile(f *models.ShareFile) error {
	return r.db.Create(f).Error
}

func (r *SharesRepository) GetFile(shareID, fileID string) (*models.ShareFile, error) {
	f := models.ShareFile{}
	return notFoundAsNil(&f, r.db.First(&f, "id = ? AND share_id = ?", fileID, shareID).Error)
}

func (r *SharesRepository) DeleteFile(shareID, fileID string) (bool, error) {
	res := r.db.Delete(&models.ShareFile{}, "id = ? AND share_id = ?", fileID, shareID)
	return res.RowsAffected > 0, res.Error
}

func (r *SharesRepository) FilesForSession(sessionID string) ([]models.ShareFile, error) {
	out := []models.ShareFile{}
	err := r.db.Where("upload_session_id = ?", sessionID).Order("uploaded_at").Find(&out).Error
	return out, err
}

// ShareUsage reports the committed bytes and file count of one Share, used for
// the per-Upload-Box caps.
func (r *SharesRepository) ShareUsage(shareID string) (bytes int64, files int, err error) {
	var row struct {
		Bytes int64
		Files int
	}
	err = r.db.Model(&models.ShareFile{}).
		Select("COALESCE(SUM(size),0) AS bytes, COUNT(*) AS files").
		Where("share_id = ?", shareID).Scan(&row).Error
	return row.Bytes, row.Files, err
}

// OwnerCommittedBytes is what the account actually has stored. This is the figure
// to show a user: it excludes uploads still in flight, so an upload that fails
// does not leave the reported usage inflated.
//
// Bytes still on disk during a Share's post-expiry grace period are included,
// which is why expiring a Share does not immediately reclaim budget.
func (r *SharesRepository) OwnerCommittedBytes(ownerID string) (int64, error) {
	var committed int64
	err := r.db.Model(&models.ShareFile{}).
		Joins("JOIN share ON share.id = share_file.share_id").
		Where("share.owner_id = ?", ownerID).
		Select("COALESCE(SUM(share_file.size),0)").Scan(&committed).Error
	return committed, err
}

// OwnerUsage adds the declared size of uploads in flight, so two concurrent
// uploads cannot both pass a check they would jointly breach. Used for
// enforcement only, never for display.
func (r *SharesRepository) OwnerUsage(ownerID string) (int64, error) {
	var committed int64
	err := r.db.Model(&models.ShareFile{}).
		Joins("JOIN share ON share.id = share_file.share_id").
		Where("share.owner_id = ?", ownerID).
		Select("COALESCE(SUM(share_file.size),0)").Scan(&committed).Error
	if err != nil {
		return 0, err
	}

	var reserved int64
	err = r.db.Model(&models.PendingUpload{}).
		Where("owner_id = ?", ownerID).
		Select("COALESCE(SUM(declared_size),0)").Scan(&reserved).Error
	if err != nil {
		return 0, err
	}

	return committed + reserved, nil
}

// UsageByOwner powers the admin storage view. It returns bytes and counts only —
// never titles or filenames — so Upload Box privacy holds against admins.
type OwnerUsageRow struct {
	OwnerID string `json:"ownerId"`
	Bytes   int64  `json:"bytes"`
	Shares  int    `json:"shares"`
}

func (r *SharesRepository) UsageByOwner() ([]OwnerUsageRow, error) {
	out := []OwnerUsageRow{}
	err := r.db.Model(&models.Share{}).
		Select("share.owner_id AS owner_id, COUNT(DISTINCT share.id) AS shares, COALESCE(SUM(share_file.size),0) AS bytes").
		Joins("LEFT JOIN share_file ON share_file.share_id = share.id").
		Group("share.owner_id").Scan(&out).Error
	return out, err
}

func (r *SharesRepository) CreatePendingUpload(p *models.PendingUpload) error {
	return r.db.Create(p).Error
}

func (r *SharesRepository) GetPendingUpload(id string) (*models.PendingUpload, error) {
	p := models.PendingUpload{}
	return notFoundAsNil(&p, r.db.First(&p, "id = ?", id).Error)
}

func (r *SharesRepository) DeletePendingUpload(id string) error {
	return r.db.Delete(&models.PendingUpload{}, "id = ?", id).Error
}

func (r *SharesRepository) PendingUploadsOlderThan(cutoff time.Time) ([]models.PendingUpload, error) {
	out := []models.PendingUpload{}
	err := r.db.Where("created_at < ?", cutoff).Find(&out).Error
	return out, err
}

func (r *SharesRepository) CountPendingForShare(shareID string) (int64, error) {
	var n int64
	err := r.db.Model(&models.PendingUpload{}).Where("share_id = ?", shareID).Count(&n).Error
	return n, err
}

// SweepableShares are those whose expiry plus the grace period has passed.
func (r *SharesRepository) SweepableShares(grace time.Duration) ([]models.Share, error) {
	out := []models.Share{}
	cutoff := time.Now().Add(-grace)
	err := r.db.Where("expires_at IS NOT NULL AND expires_at < ?", cutoff).Find(&out).Error
	return out, err
}

// EmptyShares have no files and nothing in flight. Drop-first creation means a
// Share can exist having never held anything; these are swept rather than kept.
func (r *SharesRepository) EmptyShares(olderThan time.Duration) ([]models.Share, error) {
	out := []models.Share{}
	cutoff := time.Now().Add(-olderThan)
	err := r.db.
		Where("created_at < ?", cutoff).
		Where("NOT EXISTS (SELECT 1 FROM share_file WHERE share_file.share_id = share.id)").
		Where("NOT EXISTS (SELECT 1 FROM pending_upload WHERE pending_upload.share_id = share.id)").
		Find(&out).Error
	return out, err
}

func (r *SharesRepository) DeleteForOwner(ownerID string) ([]string, error) {
	shares := []models.Share{}
	if err := r.db.Where("owner_id = ?", ownerID).Find(&shares).Error; err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(shares))
	for _, s := range shares {
		ids = append(ids, s.ID)
	}
	if len(ids) == 0 {
		return ids, nil
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.ShareFile{}, "share_id IN ?", ids).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.PendingUpload{}, "share_id IN ?", ids).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Share{}, "owner_id = ?", ownerID).Error
	})
	return ids, err
}

func (r *SharesRepository) SetNotifyError(id, msg string) error {
	now := time.Now()
	fields := map[string]any{"notify_last_error": msg, "notify_last_error_at": &now}
	if msg == "" {
		fields["notify_last_error_at"] = nil
	}
	return r.db.Model(&models.Share{}).Where("id = ?", id).Updates(fields).Error
}
