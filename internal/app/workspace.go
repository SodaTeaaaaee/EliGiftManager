package app

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type Workspace struct {
	Store         domain.Store
	Now           func() time.Time
	NewTrackingID func() (string, error)
}

func NewWorkspace(store domain.Store) *Workspace {
	return &Workspace{
		Store:         store,
		Now:           time.Now,
		NewTrackingID: RandomTrackingID,
	}
}

func RandomTrackingID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func NormalizeIdentity(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}
