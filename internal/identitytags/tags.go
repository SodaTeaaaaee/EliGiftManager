package identitytags

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

// IdentityTagEntry is the JSON structure for wave.level_tags entries.
type IdentityTagEntry struct {
	Platform  string `json:"platform"`
	TagName   string `json:"tagName,omitempty"`
	MatchMode string `json:"matchMode"`
}

// NormalizeIdentityTag validates and normalises the three fields of an identity tag.
// Allowed matchMode values are gift_level, platform_all, and wave_all.
// user_member and any unknown matchMode are rejected with an error.
//
//	gift_level   — platform and tagName must both be non-empty
//	platform_all — platform must be non-empty; tagName is forced to ""
//	wave_all     — platform and tagName are both forced to ""
//
// An empty matchMode defaults to gift_level.
func NormalizeIdentityTag(platform, tagName, matchMode string) (string, string, string, error) {
	platform = strings.TrimSpace(platform)
	tagName = strings.TrimSpace(tagName)
	matchMode = strings.TrimSpace(matchMode)
	if matchMode == "" {
		matchMode = "gift_level"
	}
	switch matchMode {
	case "gift_level":
		if platform == "" || tagName == "" {
			return "", "", "", fmt.Errorf("gift_level requires non-empty platform and tagName")
		}
	case "platform_all":
		if platform == "" {
			return "", "", "", fmt.Errorf("platform_all requires non-empty platform")
		}
		tagName = ""
	case "wave_all":
		platform = ""
		tagName = ""
	default:
		return "", "", "", fmt.Errorf("invalid identity matchMode %q: allowed values are gift_level, platform_all, wave_all", matchMode)
	}
	return platform, tagName, matchMode, nil
}

// BuildWaveIdentityTagCandidates rebuilds wave.level_tags from wave_members only:
//
//	(a) gift_level entries — one per distinct (platform, giftLevel) pair
//	(b) platform_all entries — one per distinct platform
//	(c) wave_all entry — one if any wave_member exists
//
// Sorted by platform ASC, matchMode ASC, tagName ASC. Returns the marshalled JSON string.
func BuildWaveIdentityTagCandidates(db *gorm.DB, waveID uint) (string, error) {
	var wms []model.WaveMember
	if err := db.Where("wave_id = ?", waveID).Find(&wms).Error; err != nil {
		return "", fmt.Errorf("build wave identity tag candidates: load wave members: %w", err)
	}

	giftLevelPairSet := map[string]bool{} // key: platform::giftLevel
	platformSet := map[string]bool{}
	for _, wm := range wms {
		p := strings.TrimSpace(wm.Platform)
		gl := strings.TrimSpace(wm.GiftLevel)
		if p != "" {
			platformSet[p] = true
			if gl != "" {
				giftLevelPairSet[p+"::"+gl] = true
			}
		}
	}

	var entries []IdentityTagEntry

	// gift_level entries.
	for key := range giftLevelPairSet {
		parts := strings.SplitN(key, "::", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, IdentityTagEntry{
			Platform:  parts[0],
			TagName:   parts[1],
			MatchMode: "gift_level",
		})
	}

	// platform_all entries.
	for platform := range platformSet {
		entries = append(entries, IdentityTagEntry{
			Platform:  platform,
			TagName:   "",
			MatchMode: "platform_all",
		})
	}

	// wave_all entry.
	if len(wms) > 0 {
		entries = append(entries, IdentityTagEntry{
			Platform:  "",
			TagName:   "",
			MatchMode: "wave_all",
		})
	}

	// Sort for deterministic output.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Platform != entries[j].Platform {
			return entries[i].Platform < entries[j].Platform
		}
		if entries[i].MatchMode != entries[j].MatchMode {
			return entries[i].MatchMode < entries[j].MatchMode
		}
		return entries[i].TagName < entries[j].TagName
	})

	b, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("build wave identity tag candidates: marshal: %w", err)
	}
	return string(b), nil
}

// ProductTagMatchesWaveMember returns true if the identity tag should match the given
// wave member. Dispatch:
//
//	gift_level   → wm.Platform == tag.Platform && wm.GiftLevel == tag.TagName
//	platform_all → wm.Platform == tag.Platform
//	wave_all     → true (matches all members in the wave)
//
// This function only considers identity tags (tag_type='identity'). Caller is
// responsible for filtering tag_type before calling. user_member is not a valid
// identity matchMode and will return false via the default branch.
func ProductTagMatchesWaveMember(tag model.ProductTag, wm model.WaveMember) bool {
	switch tag.MatchMode {
	case "gift_level":
		return wm.Platform == tag.Platform && wm.GiftLevel == tag.TagName
	case "platform_all":
		return wm.Platform == tag.Platform
	case "wave_all":
		return true
	default:
		return false
	}
}

// CalculateWaveMemberIdentityBaseQuantity sums identity tag quantities that match the
// given wave member for a specific product. Only looks at tag_type='identity' tags.
// Each tag whose ProductTagMatchesWaveMember returns true contributes its quantity.
func CalculateWaveMemberIdentityBaseQuantity(db *gorm.DB, productID uint, wm model.WaveMember) int {
	var tags []model.ProductTag
	db.Where("product_id = ? AND tag_type = 'identity'", productID).Find(&tags)
	total := 0
	for _, tag := range tags {
		if ProductTagMatchesWaveMember(tag, wm) {
			total += tag.Quantity
		}
	}
	return total
}
