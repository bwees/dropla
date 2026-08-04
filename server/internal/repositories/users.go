package repositories

import (
	"errors"
	"time"

	"app/internal/database/models"

	"gorm.io/gorm"
)

type UsersRepository struct {
	db *gorm.DB
}

func NewUsersRepository(db *gorm.DB) *UsersRepository {
	return &UsersRepository{db: db}
}

// notFoundAsNil lets callers distinguish "absent" from "failed" without
// inspecting driver errors at every call site.
func notFoundAsNil[T any](out *T, err error) (*T, error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

func (r *UsersRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UsersRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *UsersRepository) List() ([]models.User, error) {
	users := []models.User{}
	err := r.db.Order("username").Find(&users).Error
	return users, err
}

func (r *UsersRepository) GetByUsername(username string) (*models.User, error) {
	user := models.User{}
	return notFoundAsNil(&user, r.db.First(&user, "username = ?", username).Error)
}

func (r *UsersRepository) GetByID(id string) (*models.User, error) {
	user := models.User{}
	return notFoundAsNil(&user, r.db.First(&user, "id = ?", id).Error)
}

func (r *UsersRepository) UpdatePassword(id, hash string) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}

func (r *UsersRepository) UpdateQuota(id string, quota *int64) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("quota_bytes", quota).Error
}

func (r *UsersRepository) Delete(id string) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}

type SessionsRepository struct {
	db *gorm.DB
}

func NewSessionsRepository(db *gorm.DB) *SessionsRepository {
	return &SessionsRepository{db: db}
}

func (r *SessionsRepository) Create(s *models.Session) error {
	return r.db.Create(s).Error
}

func (r *SessionsRepository) Get(tokenHash string) (*models.Session, error) {
	s := models.Session{}
	return notFoundAsNil(&s, r.db.First(&s, "token_hash = ?", tokenHash).Error)
}

func (r *SessionsRepository) Delete(tokenHash string) error {
	return r.db.Delete(&models.Session{}, "token_hash = ?", tokenHash).Error
}

func (r *SessionsRepository) DeleteForUser(userID string) error {
	return r.db.Delete(&models.Session{}, "user_id = ?", userID).Error
}

func (r *SessionsRepository) DeleteExpired() error {
	return r.db.Delete(&models.Session{}, "expires_at < ?", time.Now()).Error
}

type AccessTokensRepository struct {
	db *gorm.DB
}

func NewAccessTokensRepository(db *gorm.DB) *AccessTokensRepository {
	return &AccessTokensRepository{db: db}
}

func (r *AccessTokensRepository) Create(t *models.AccessToken) error {
	return r.db.Create(t).Error
}

func (r *AccessTokensRepository) ListForUser(userID string) ([]models.AccessToken, error) {
	out := []models.AccessToken{}
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&out).Error
	return out, err
}

func (r *AccessTokensRepository) GetByHash(hash string) (*models.AccessToken, error) {
	t := models.AccessToken{}
	return notFoundAsNil(&t, r.db.First(&t, "token_hash = ?", hash).Error)
}

func (r *AccessTokensRepository) TouchLastUsed(id string) error {
	now := time.Now()
	return r.db.Model(&models.AccessToken{}).Where("id = ?", id).Update("last_used_at", now).Error
}

func (r *AccessTokensRepository) Delete(userID, id string) (bool, error) {
	res := r.db.Delete(&models.AccessToken{}, "id = ? AND user_id = ?", id, userID)
	return res.RowsAffected > 0, res.Error
}

func (r *AccessTokensRepository) DeleteForUser(userID string) error {
	return r.db.Delete(&models.AccessToken{}, "user_id = ?", userID).Error
}
