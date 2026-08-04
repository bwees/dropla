package controllers

import (
	"context"
	"net/http"
	"strings"

	"app/internal/database/models"
	"app/internal/services"
)

const sessionCookieName = "session"

type contextKey string

const userContextKey contextKey = "user"

// publicPaths are exact /api paths reachable without a session.
var publicPaths = map[string]bool{
	"/api/auth/setup":  true,
	"/api/auth/login":  true,
	"/api/auth/logout": true,
}

// publicPrefixes are /api subtrees that authorise themselves rather than by
// session: Share links carry their own capability, tus uses the upload id as a
// bearer secret, and the CLI presents an access token.
var publicPrefixes = []string{
	"/api/s/",
	"/api/uploads/",
	"/api/cli/",
}

type AuthMiddleware struct {
	auth *services.AuthService
}

func NewAuthMiddleware(auth *services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{auth: auth}
}

// Handler requires a valid session for /api routes, letting non-API paths (the
// static frontend and the /s/ download endpoints) and the self-authorising
// subtrees through.
//
// It deliberately passes the ResponseWriter through untouched. Wrapping it
// without an Unwrap method defeats http.ResponseController and io.ReaderFrom,
// which would silently break resumable uploads and sendfile on downloads.
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.requiresSession(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := m.auth.Authenticate(cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

func (m *AuthMiddleware) requiresSession(path string) bool {
	if !strings.HasPrefix(path, "/api") {
		return false
	}
	if publicPaths[path] {
		return false
	}
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return true
}

func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFrom(ctx context.Context) *models.User {
	user, _ := ctx.Value(userContextKey).(*models.User)
	return user
}

func requireUser(ctx context.Context) (*models.User, error) {
	user := UserFrom(ctx)
	if user == nil {
		return nil, services.ErrInvalidCredentials
	}
	return user, nil
}

func requireAdmin(ctx context.Context) (*models.User, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin {
		return nil, services.ErrForbidden
	}
	return user, nil
}

func sessionCookie(token string, secure bool, maxAge int) http.Cookie {
	return http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

// shareUnlockCookie remembers that a visitor entered a Share's password.
//
// It is keyed on the Share's internal id rather than its link token, so the
// cookie name never embeds the link secret and the grant survives link rotation
// — rotation revokes URLs, a password change revokes unlocks, and the two are
// deliberately independent levers. Path=/ is required because Share metadata
// lives under /api/s/ while downloads live under /s/.
func shareUnlockCookie(shareID, unlockToken string, secure bool, maxAge int) http.Cookie {
	return http.Cookie{
		Name:     "dropla_s_" + shareID,
		Value:    unlockToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

func unlockCookieValue(r *http.Request, shareID string) string {
	cookie, err := r.Cookie("dropla_s_" + shareID)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func isSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
