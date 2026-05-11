package service

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/identitytags"
	"gorm.io/gorm"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// IdentityTagEntry is the JSON structure for wave.level_tags entries.
// Re-exported from the shared identitytags package.
type IdentityTagEntry = identitytags.IdentityTagEntry

// NormalizeIdentityTag validates and normalises identity tag fields.
// Re-exported from the shared identitytags package.
func NormalizeIdentityTag(platform, tagName, matchMode string) (string, string, string, error) {
	return identitytags.NormalizeIdentityTag(platform, tagName, matchMode)
}

// BuildWaveIdentityTagCandidates rebuilds wave.level_tags from wave_member-derived
// gift_level, platform_all, and wave_all entries.
func BuildWaveIdentityTagCandidates(db *gorm.DB, waveID uint) (string, error) {
	return identitytags.BuildWaveIdentityTagCandidates(db, waveID)
}

// ProductTagMatchesWaveMember returns true if the identity tag should match the given
// wave member per its matchMode.
func ProductTagMatchesWaveMember(tag model.ProductTag, wm model.WaveMember) bool {
	return identitytags.ProductTagMatchesWaveMember(tag, wm)
}

// CalculateWaveMemberIdentityBaseQuantity sums identity tag quantities matching the
// given wave member for a specific product.
func CalculateWaveMemberIdentityBaseQuantity(db *gorm.DB, productID uint, wm model.WaveMember) int {
	return identitytags.CalculateWaveMemberIdentityBaseQuantity(db, productID, wm)
}
