package services

import (
	"strings"
	"time"

	"app/internal/config"
	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/repositories"
	"app/internal/tokens"

	"golang.org/x/crypto/bcrypt"
)

const SessionDuration = 30 * 24 * time.Hour

type AuthService struct {
	users    *repositories.UsersRepository
	sessions *repositories.SessionsRepository
	access   *repositories.AccessTokensRepository
	shares   *repositories.SharesRepository
}

func NewAuthService(
	users *repositories.UsersRepository,
	sessions *repositories.SessionsRepository,
	access *repositories.AccessTokensRepository,
	shares *repositories.SharesRepository,
) *AuthService {
	return &AuthService{users: users, sessions: sessions, access: access, shares: shares}
}

func (s *AuthService) SetupRequired() (bool, error) {
	count, err := s.users.Count()
	return count == 0, err
}

func (s *AuthService) Setup(dto dtos.CreateUserDto) (*models.User, error) {
	required, err := s.SetupRequired()
	if err != nil {
		return nil, err
	}
	if !required {
		return nil, ErrSetupComplete
	}
	return s.createUser(dto.Username, dto.Password, true)
}

func (s *AuthService) CreateUser(dto dtos.CreateUserDto) (*models.User, error) {
	return s.createUser(dto.Username, dto.Password, dto.IsAdmin)
}

func (s *AuthService) createUser(username, password string, isAdmin bool) (*models.User, error) {
	username = strings.TrimSpace(username)

	existing, err := s.users.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{Username: username, PasswordHash: string(hash), IsAdmin: isAdmin}
	if err := s.users.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) ListUsers() ([]models.User, error) { return s.users.List() }

// DeleteUser cascades to the account's Shares, their files and their blobs.
// Links the user handed out stop working, and none of it is recoverable — the
// caller is expected to have warned first. Blob removal is the caller's job via
// the returned Share ids, since this layer does not touch the filesystem.
func (s *AuthService) DeleteUser(id string) ([]string, error) {
	shareIDs, err := s.shares.DeleteForOwner(id)
	if err != nil {
		return nil, err
	}
	if err := s.sessions.DeleteForUser(id); err != nil {
		return nil, err
	}
	if err := s.access.DeleteForUser(id); err != nil {
		return nil, err
	}
	return shareIDs, s.users.Delete(id)
}

func (s *AuthService) Login(dto dtos.LoginDto) (*models.User, string, error) {
	user, err := s.users.GetByUsername(strings.TrimSpace(dto.Username))
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		// Hash a throwaway value so a missing username and a wrong password cost
		// the same time.
		_, _ = bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		return nil, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.Password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token := tokens.Session()
	session := &models.Session{
		TokenHash: tokens.Hash(token),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(SessionDuration),
	}
	if err := s.sessions.Create(session); err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *AuthService) Logout(token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Delete(tokens.Hash(token))
}

func (s *AuthService) Authenticate(token string) (*models.User, error) {
	if token == "" {
		return nil, ErrInvalidCredentials
	}

	session, err := s.sessions.Get(tokens.Hash(token))
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrInvalidCredentials
	}
	if time.Now().After(session.ExpiresAt) {
		_ = s.sessions.Delete(session.TokenHash)
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.GetByID(session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

// ChangeOwnPassword requires the current password. Like a reset, it drops every
// session for the user: people change passwords because they suspect compromise,
// so a change that left an attacker's session alive would be a false promise.
func (s *AuthService) ChangeOwnPassword(userID string, dto dtos.SetPasswordDto) error {
	user, err := s.users.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.CurrentPassword)); err != nil {
		return ErrInvalidCredentials
	}
	return s.setPassword(userID, dto.NewPassword)
}

// ResetPassword is the admin path and needs no current password. Admins may
// reset anyone including other admins: they can already delete any account, so
// withholding this would guard a lesser power than one they hold.
func (s *AuthService) ResetPassword(userID, newPassword string) error {
	user, err := s.users.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}
	return s.setPassword(userID, newPassword)
}

func (s *AuthService) setPassword(userID, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(userID, string(hash)); err != nil {
		return err
	}
	return s.sessions.DeleteForUser(userID)
}

func (s *AuthService) SetQuota(userID string, quota *int64) error {
	return s.users.UpdateQuota(userID, quota)
}

// CreateAccessToken returns the secret exactly once, inlined into a runnable
// command. Nothing re-displays it, which is the point of storing only the hash.
func (s *AuthService) CreateAccessToken(userID, name string) (*dtos.AccessTokenCreatedDto, error) {
	secret := tokens.Access()
	record := &models.AccessToken{
		ID:        uuidString(),
		UserID:    userID,
		Name:      strings.TrimSpace(name),
		TokenHash: tokens.Hash(secret),
	}
	if err := s.access.Create(record); err != nil {
		return nil, err
	}

	base := config.PublicBaseURL()
	if base == "" {
		base = "http://localhost:" + config.Port()
	}

	return &dtos.AccessTokenCreatedDto{
		ID:    record.ID,
		Name:  record.Name,
		Token: secret,
		Command: "curl -T ./file.zip \\\n" +
			"  -H \"Authorization: Bearer " + secret + "\" \\\n  " +
			base + "/api/cli/upload | jq -r .url",
	}, nil
}

func (s *AuthService) ListAccessTokens(userID string) ([]models.AccessToken, error) {
	return s.access.ListForUser(userID)
}

func (s *AuthService) DeleteAccessToken(userID, id string) error {
	deleted, err := s.access.Delete(userID, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

// AuthenticateAccessToken resolves a CLI bearer token to its owner.
func (s *AuthService) AuthenticateAccessToken(secret string) (*models.User, error) {
	if secret == "" {
		return nil, ErrInvalidCredentials
	}

	record, err := s.access.GetByHash(tokens.Hash(secret))
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.GetByID(record.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	_ = s.access.TouchLastUsed(record.ID)
	return user, nil
}

func (s *AuthService) QuotaFor(user *models.User) int64 {
	if user.QuotaBytes != nil {
		return *user.QuotaBytes
	}
	return config.DefaultUserQuotaBytes()
}
