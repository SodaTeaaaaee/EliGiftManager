package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// maxWritebackErrorRunes caps the stored error text so a runaway platform
// response cannot bloat the writeback row.
const maxWritebackErrorRunes = 500

// MarkWritebackSent records that the parcel's writeback reached the source
// platform: status becomes sent and the error text of the last failed attempt
// is cleared. RetryCount is deliberately kept so the retry history survives a
// later success.
func (ws *Workspace) MarkWritebackSent(ctx context.Context, writebackID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		wb, err := tx.GetWriteback(ctx, writebackID)
		if err != nil {
			return err
		}
		wb.Status = string(domain.WritebackSent)
		wb.ErrorMessage = ""
		return tx.UpdateWriteback(ctx, wb)
	})
}

// MarkWritebackFailed records a failed writeback attempt: status becomes
// failed, the retry counter advances, and the error text of this attempt is
// stored (truncated to maxWritebackErrorRunes runes).
func (ws *Workspace) MarkWritebackFailed(ctx context.Context, writebackID uint, errMsg string) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		wb, err := tx.GetWriteback(ctx, writebackID)
		if err != nil {
			return err
		}
		wb.Status = string(domain.WritebackFailed)
		wb.RetryCount++
		wb.ErrorMessage = truncateRunes(errMsg, maxWritebackErrorRunes)
		return tx.UpdateWriteback(ctx, wb)
	})
}

// truncateRunes shortens s to at most n runes without splitting a character.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
