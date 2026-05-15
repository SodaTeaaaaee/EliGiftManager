package selector

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func makeParticipants() []domain.WaveParticipantSnapshot {
	return []domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: 10, IdentityPlatform: "bilibili", GiftLevel: "gold", DisplayName: "Alice"},
		{ID: 2, WaveID: 10, IdentityPlatform: "bilibili", GiftLevel: "silver", DisplayName: "Bob"},
		{ID: 3, WaveID: 10, IdentityPlatform: "douyin", GiftLevel: "gold", DisplayName: "Carol"},
		{ID: 4, WaveID: 10, IdentityPlatform: "douyin", GiftLevel: "bronze", DisplayName: "Dave"},
	}
}

func TestMatchSelector_WaveAll(t *testing.T) {
	participants := makeParticipants()
	payload := domain.SelectorPayload{Type: "wave_all"}

	result := MatchSelector(payload, participants)

	if len(result) != len(participants) {
		t.Errorf("wave_all: expected %d participants, got %d", len(participants), len(result))
	}
}

func TestMatchSelector_PlatformAll(t *testing.T) {
	participants := makeParticipants()
	payload := domain.SelectorPayload{Type: "platform_all", Platform: "bilibili"}

	result := MatchSelector(payload, participants)

	if len(result) != 2 {
		t.Errorf("platform_all: expected 2 participants, got %d", len(result))
	}
	for _, p := range result {
		if p.IdentityPlatform != "bilibili" {
			t.Errorf("platform_all: unexpected platform %q", p.IdentityPlatform)
		}
	}
}

func TestMatchSelector_IdentityLevel(t *testing.T) {
	participants := makeParticipants()
	payload := domain.SelectorPayload{Type: "identity_level", Platform: "bilibili", Level: "gold"}

	result := MatchSelector(payload, participants)

	if len(result) != 1 {
		t.Errorf("identity_level: expected 1 participant, got %d", len(result))
	}
	if len(result) > 0 && result[0].DisplayName != "Alice" {
		t.Errorf("identity_level: expected Alice, got %q", result[0].DisplayName)
	}
}

func TestMatchSelector_ExplicitOverride(t *testing.T) {
	participants := makeParticipants()
	payload := domain.SelectorPayload{Type: "explicit_override", ParticipantIDs: []uint{2, 4}}

	result := MatchSelector(payload, participants)

	if len(result) != 2 {
		t.Errorf("explicit_override: expected 2 participants, got %d", len(result))
	}
	ids := map[uint]bool{}
	for _, p := range result {
		ids[p.ID] = true
	}
	if !ids[2] || !ids[4] {
		t.Errorf("explicit_override: expected IDs {2,4}, got %v", ids)
	}
}

func TestMatchSelector_EmptyParticipants(t *testing.T) {
	payload := domain.SelectorPayload{Type: "wave_all"}

	result := MatchSelector(payload, nil)

	if result != nil {
		t.Errorf("empty participants with wave_all: expected nil, got %v", result)
	}
}

func TestMatchSelector_UnknownType(t *testing.T) {
	participants := makeParticipants()
	payload := domain.SelectorPayload{Type: "nonexistent_type"}

	result := MatchSelector(payload, participants)

	if len(result) != 0 {
		t.Errorf("unknown type: expected 0 participants, got %d", len(result))
	}
}
