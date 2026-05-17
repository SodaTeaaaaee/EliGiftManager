package app

import (
	"encoding/json"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// CaptureProfileSnapshot serializes the execution-relevant fields of a live profile
// into a JSON string suitable for storage on DemandDocument.BoundProfileSnapshot.
func CaptureProfileSnapshot(profile *domain.IntegrationProfile) string {
	snap := dto.BoundProfileSnapshot{
		ProfileID:               profile.ID,
		ProfileKey:              profile.ProfileKey,
		TrackingSyncMode:        profile.TrackingSyncMode,
		ClosurePolicy:           profile.ClosurePolicy,
		AllowsManualClosure:     profile.AllowsManualClosure,
		RequiresCarrierMapping:  profile.RequiresCarrierMapping,
		RequiresExternalOrderNo: profile.RequiresExternalOrderNo,
		SupportsPartialShipment: profile.SupportsPartialShipment,
		ConnectorKey:            profile.ConnectorKey,
		SupportsAPIExport:       profile.SupportsAPIExport,
	}
	data, _ := json.Marshal(snap)
	return string(data)
}

// ParseProfileSnapshot deserializes a BoundProfileSnapshot from its JSON string form.
// Returns nil, nil when raw is empty (no snapshot stored yet).
func ParseProfileSnapshot(raw string) (*dto.BoundProfileSnapshot, error) {
	if raw == "" {
		return nil, nil
	}
	var snap dto.BoundProfileSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// ResolveEffectiveProfile returns the bound snapshot when one is stored on the document,
// falling back to a live profile lookup for backward compatibility with pre-binding data.
// Callers should prefer the snapshot path; the live fallback exists only for documents
// that were assigned before this feature was introduced.
func ResolveEffectiveProfile(doc *domain.DemandDocument, profileRepo domain.IntegrationProfileRepository) (*dto.BoundProfileSnapshot, error) {
	// Prefer the bound snapshot — this is the stable, wave-time view of the profile.
	if doc.BoundProfileSnapshot != "" {
		snap, err := ParseProfileSnapshot(doc.BoundProfileSnapshot)
		if err == nil && snap != nil {
			return snap, nil
		}
		// Corrupt snapshot: fall through to live lookup rather than hard-failing.
	}

	// Fallback: live profile lookup for documents assigned before snapshot capture was introduced.
	if doc.IntegrationProfileID == nil {
		return nil, nil
	}
	profile, err := profileRepo.FindByID(*doc.IntegrationProfileID)
	if err != nil {
		return nil, err
	}
	snap := &dto.BoundProfileSnapshot{
		ProfileID:               profile.ID,
		ProfileKey:              profile.ProfileKey,
		TrackingSyncMode:        profile.TrackingSyncMode,
		ClosurePolicy:           profile.ClosurePolicy,
		AllowsManualClosure:     profile.AllowsManualClosure,
		RequiresCarrierMapping:  profile.RequiresCarrierMapping,
		RequiresExternalOrderNo: profile.RequiresExternalOrderNo,
		SupportsPartialShipment: profile.SupportsPartialShipment,
		ConnectorKey:            profile.ConnectorKey,
		SupportsAPIExport:       profile.SupportsAPIExport,
	}
	return snap, nil
}
