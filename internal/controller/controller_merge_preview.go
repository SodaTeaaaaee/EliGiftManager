package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// PreviewMergeProfiles computes a read-only conflict-detail preview for a
// prospective MergeProfiles call — both sides' identities/addresses plus a
// highlighted conflict list — so the merge dialog can show it before the
// operator commits (plan 5.2). Unlike MergeProfiles, this never mutates data
// so it does not need a transaction.
func (c *MergeController) PreviewMergeProfiles(sourceProfileID, targetProfileID uint) (*dto.MergeProfilesPreviewResult, error) {
	ctx := appContext
	profileRepo := infra.NewProfileRepository(c.gdb)
	addressRepo := infra.NewAddressRepository(c.gdb)
	previewUC := app.NewProfileMergePreviewUseCase(profileRepo, addressRepo)
	return previewUC.PreviewMergeProfiles(ctx, sourceProfileID, targetProfileID)
}
