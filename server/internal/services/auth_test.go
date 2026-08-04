package services

import (
	"errors"
	"testing"

	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/repositories"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAuthService(t *testing.T) (*AuthService, *repositories.SessionsRepository) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Session{}, &models.AccessToken{},
		&models.Share{}, &models.ShareFile{}, &models.PendingUpload{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sessions := repositories.NewSessionsRepository(db)
	svc := NewAuthService(
		repositories.NewUsersRepository(db),
		sessions,
		repositories.NewAccessTokensRepository(db),
		repositories.NewSharesRepository(db),
	)
	return svc, sessions
}

func TestSetupOnlyRunsOnce(t *testing.T) {
	svc, _ := newAuthService(t)

	required, err := svc.SetupRequired()
	if err != nil || !required {
		t.Fatalf("expected setup to be required, got %v (err %v)", required, err)
	}

	first, err := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !first.IsAdmin {
		t.Fatal("the first account must be an admin")
	}

	if _, err := svc.Setup(dtos.CreateUserDto{Username: "other", Password: "correcthorse"}); !errors.Is(err, ErrSetupComplete) {
		t.Fatalf("expected ErrSetupComplete on second setup, got %v", err)
	}
}

func TestLoginRejectsWrongPasswordAndUnknownUser(t *testing.T) {
	svc, _ := newAuthService(t)
	if _, err := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if _, _, err := svc.Login(dtos.LoginDto{Username: "bwees", Password: "wrong"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}
	if _, _, err := svc.Login(dtos.LoginDto{Username: "ghost", Password: "correcthorse"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user: got %v", err)
	}

	_, token, err := svc.Login(dtos.LoginDto{Username: "bwees", Password: "correcthorse"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := svc.Authenticate(token); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
}

// A password change must invalidate every session, including ones on other
// devices: people change passwords because they suspect compromise.
func TestPasswordChangeDropsAllSessions(t *testing.T) {
	svc, _ := newAuthService(t)
	user, err := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, laptop, _ := svc.Login(dtos.LoginDto{Username: "bwees", Password: "correcthorse"})
	_, phone, _ := svc.Login(dtos.LoginDto{Username: "bwees", Password: "correcthorse"})

	err = svc.ChangeOwnPassword(user.ID, dtos.SetPasswordDto{
		CurrentPassword: "correcthorse", NewPassword: "batterystaple",
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	for name, token := range map[string]string{"laptop": laptop, "phone": phone} {
		if _, err := svc.Authenticate(token); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("%s session survived a password change: %v", name, err)
		}
	}

	if _, _, err := svc.Login(dtos.LoginDto{Username: "bwees", Password: "batterystaple"}); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestChangeOwnPasswordRequiresCurrent(t *testing.T) {
	svc, _ := newAuthService(t)
	user, _ := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"})

	err := svc.ChangeOwnPassword(user.ID, dtos.SetPasswordDto{
		CurrentPassword: "guess", NewPassword: "batterystaple",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

// An admin reset needs no current password, and also signs the target out.
func TestAdminResetDropsSessions(t *testing.T) {
	svc, _ := newAuthService(t)
	if _, err := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	alice, err := svc.CreateUser(dtos.CreateUserDto{Username: "alice", Password: "hunter2222"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, aliceToken, _ := svc.Login(dtos.LoginDto{Username: "alice", Password: "hunter2222"})

	if err := svc.ResetPassword(alice.ID, "freshpassword"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := svc.Authenticate(aliceToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("session survived an admin reset")
	}
}

func TestUsernamesAreUnique(t *testing.T) {
	svc, _ := newAuthService(t)
	if _, err := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := svc.CreateUser(dtos.CreateUserDto{Username: "bwees", Password: "otherpass1"}); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestAccessTokenRoundTripAndRevocation(t *testing.T) {
	svc, _ := newAuthService(t)
	user, _ := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"})

	created, err := svc.CreateAccessToken(user.ID, "laptop")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if created.Token == "" || created.Command == "" {
		t.Fatal("a created token must return its secret and a runnable command once")
	}

	resolved, err := svc.AuthenticateAccessToken(created.Token)
	if err != nil || resolved.ID != user.ID {
		t.Fatalf("authenticate token: %v", err)
	}

	if err := svc.DeleteAccessToken(user.ID, created.ID); err != nil {
		t.Fatalf("delete token: %v", err)
	}
	if _, err := svc.AuthenticateAccessToken(created.Token); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("revoked token still authenticates")
	}
}

// A CLI token is revocable on its own; that is the whole reason it is not the
// account password, since a password change would sign the user out everywhere.
func TestAccessTokenSurvivesPasswordChange(t *testing.T) {
	svc, _ := newAuthService(t)
	user, _ := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"})

	created, _ := svc.CreateAccessToken(user.ID, "ci")
	if err := svc.ChangeOwnPassword(user.ID, dtos.SetPasswordDto{
		CurrentPassword: "correcthorse", NewPassword: "batterystaple",
	}); err != nil {
		t.Fatalf("change password: %v", err)
	}

	if _, err := svc.AuthenticateAccessToken(created.Token); err != nil {
		t.Fatalf("CLI token should survive a password change: %v", err)
	}
}

func TestDeleteUserRemovesTokensAndSessions(t *testing.T) {
	svc, _ := newAuthService(t)
	if _, err := svc.Setup(dtos.CreateUserDto{Username: "bwees", Password: "correcthorse"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	alice, _ := svc.CreateUser(dtos.CreateUserDto{Username: "alice", Password: "hunter2222"})
	_, aliceToken, _ := svc.Login(dtos.LoginDto{Username: "alice", Password: "hunter2222"})
	cli, _ := svc.CreateAccessToken(alice.ID, "laptop")

	if _, err := svc.DeleteUser(alice.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	if _, err := svc.Authenticate(aliceToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("session survived account deletion")
	}
	if _, err := svc.AuthenticateAccessToken(cli.Token); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("CLI token survived account deletion")
	}
}
