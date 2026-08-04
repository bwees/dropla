// Package tokens mints and hashes the opaque secrets dropla hands out.
//
// Identity and capability are deliberately separate concerns: a Share's UUID is
// its identity and grants nothing, while its access token is the capability that
// appears in public URLs. Keeping them apart means the internal id is safe to
// log, and a leaked link can be revoked by minting a new token without touching
// the row's identity.
package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const (
	shareBytes   = 16
	sessionBytes = 32
	accessPrefix = "dpla_"
)

func random(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("tokens: crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Share mints the 22-character capability that appears in /s/<token> URLs.
func Share() string { return random(shareBytes) }

// Unlock mints the value stored in a share-unlock cookie. It is regenerated
// whenever a Share's password changes, which is what makes changing the password
// revoke every outstanding unlock.
func Unlock() string { return random(shareBytes) }

func Session() string { return random(sessionBytes) }

// Access mints a CLI token. The prefix makes it recognisable in a log or a
// secret-scanning report.
func Access() string { return accessPrefix + random(sessionBytes) }

// Hash is used for anything stored at rest rather than compared in the clear.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
