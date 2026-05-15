package selector

import "github.com/SodaTeaaaaee/EliGiftManager/internal/domain"

// strategyMap dispatches selector matching by SelectorPayload.Type.
var strategyMap = map[string]func(domain.SelectorPayload, []domain.WaveParticipantSnapshot) []domain.WaveParticipantSnapshot{
	"wave_all":          matchWaveAll,
	"platform_all":     matchPlatformAll,
	"identity_level":   matchIdentityLevel,
	"explicit_override": matchExplicitOverride,
}

// MatchSelector filters participants according to the selector payload.
// Unknown selector types return an empty slice.
func MatchSelector(payload domain.SelectorPayload, participants []domain.WaveParticipantSnapshot) []domain.WaveParticipantSnapshot {
	fn, ok := strategyMap[payload.Type]
	if !ok {
		return []domain.WaveParticipantSnapshot{}
	}
	return fn(payload, participants)
}

func matchWaveAll(_ domain.SelectorPayload, participants []domain.WaveParticipantSnapshot) []domain.WaveParticipantSnapshot {
	return participants
}

func matchPlatformAll(payload domain.SelectorPayload, participants []domain.WaveParticipantSnapshot) []domain.WaveParticipantSnapshot {
	result := make([]domain.WaveParticipantSnapshot, 0, len(participants))
	for _, p := range participants {
		if p.IdentityPlatform == payload.Platform {
			result = append(result, p)
		}
	}
	return result
}

func matchIdentityLevel(payload domain.SelectorPayload, participants []domain.WaveParticipantSnapshot) []domain.WaveParticipantSnapshot {
	result := make([]domain.WaveParticipantSnapshot, 0, len(participants))
	for _, p := range participants {
		if p.IdentityPlatform == payload.Platform && p.GiftLevel == payload.Level {
			result = append(result, p)
		}
	}
	return result
}

func matchExplicitOverride(payload domain.SelectorPayload, participants []domain.WaveParticipantSnapshot) []domain.WaveParticipantSnapshot {
	idSet := make(map[uint]struct{}, len(payload.ParticipantIDs))
	for _, id := range payload.ParticipantIDs {
		idSet[id] = struct{}{}
	}
	result := make([]domain.WaveParticipantSnapshot, 0, len(payload.ParticipantIDs))
	for _, p := range participants {
		if _, ok := idSet[p.ID]; ok {
			result = append(result, p)
		}
	}
	return result
}
