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

// withStore returns a shallow copy of the workspace bound to another store
// (typically a transaction). The store pool is limited to a single connection,
// so inside a WithTx callback every read and write must go through the
// transaction-backed workspace, never through the outer one.
func (ws *Workspace) withStore(s domain.Store) *Workspace {
	return &Workspace{Store: s, Now: ws.Now, NewTrackingID: ws.NewTrackingID}
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
