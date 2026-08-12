package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

const (
	customerNameEventObserved          = "observed"
	customerNameEventProjectionChanged = "display_name_changed"
	customerNameEventPinned            = "pinned"
	customerNameEventUnpinned          = "unpinned"
)

var (
	ErrCustomerProfileStale  = errors.New("customer profile row version is stale")
	ErrCustomerProfileMerged = errors.New("merged customer profile is read-only")
)

type ObserveCustomerNameInput struct {
	CustomerProfileID          uint
	Name                       string
	NameKind                   string
	Authority                  string
	TrustScore                 float64
	SourceEventKey             string
	SourceIntegrationProfileID *uint
	SourceDocumentID           *uint
	SourceIdentityID           *uint
	ObservedAt                 time.Time
	ExtraData                  string
}

type ObserveCustomerNameResult struct {
	Observation        *domain.CustomerNameObservation
	Created            bool
	DisplayNameChanged bool
}

type CustomerNameObservationService struct {
	profiles     domain.CustomerProfileRepository
	displayNames domain.CustomerDisplayNameRepository
	observations domain.CustomerNameObservationRepository
	events       domain.CustomerNameEventRepository
}

func NewCustomerNameObservationService(
	profiles domain.CustomerProfileRepository,
	observations domain.CustomerNameObservationRepository,
	events domain.CustomerNameEventRepository,
) *CustomerNameObservationService {
	displayNames, _ := profiles.(domain.CustomerDisplayNameRepository)
	return &CustomerNameObservationService{
		profiles: profiles, displayNames: displayNames, observations: observations, events: events,
	}
}

// Observe claims the raw source event first, then rebuilds chronological name
// episodes from raw observed events. Callers must supply transaction-bound
// repositories when the observation is part of a larger import.
func (s *CustomerNameObservationService) Observe(ctx context.Context, input ObserveCustomerNameInput) (*ObserveCustomerNameResult, error) {
	return s.observe(ctx, input, true)
}

func (s *CustomerNameObservationService) observe(ctx context.Context, input ObserveCustomerNameInput, applyProjection bool) (*ObserveCustomerNameResult, error) {
	if err := requireCustomerResolutionFeature(ctx, s.profiles, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	if input.CustomerProfileID == 0 {
		return nil, fmt.Errorf("observe customer name: customer profile ID is required")
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, fmt.Errorf("observe customer name: name is required")
	}
	input.SourceEventKey = strings.TrimSpace(input.SourceEventKey)
	if input.SourceEventKey == "" {
		return nil, fmt.Errorf("observe customer name: source event key is required")
	}
	if input.NameKind == "" {
		input.NameKind = domain.CustomerNameKindTrustedNickname
	}
	if input.ObservedAt.IsZero() {
		input.ObservedAt = time.Now().UTC()
	} else {
		input.ObservedAt = input.ObservedAt.UTC()
	}
	payload, err := json.Marshal(domain.CustomerNameEventPayload{
		NameKind: input.NameKind, Authority: input.Authority, TrustScore: input.TrustScore,
		SourceIntegrationProfileID: input.SourceIntegrationProfileID, SourceDocumentID: input.SourceDocumentID,
		SourceIdentityID: input.SourceIdentityID, ExtraData: input.ExtraData,
	})
	if err != nil {
		return nil, fmt.Errorf("observe customer name: encode event payload: %w", err)
	}
	event := &domain.CustomerNameEvent{
		EventKey: input.SourceEventKey, CustomerProfileID: input.CustomerProfileID,
		EventKind: customerNameEventObserved, NewName: input.Name, ReasonCode: input.NameKind,
		ActorRef: input.Authority, Payload: string(payload), CreatedAt: input.ObservedAt,
	}
	created, err := s.events.CreateIfAbsent(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("observe customer name: claim event: %w", err)
	}
	if !created {
		var observation *domain.CustomerNameObservation
		if event.ObservationID != nil {
			observation, _ = s.observations.FindByID(ctx, *event.ObservationID)
		}
		return &ObserveCustomerNameResult{Observation: observation, Created: false}, nil
	}

	observation, preferred, err := s.rebuildEpisodes(ctx, input.CustomerProfileID, event.ID)
	if err != nil {
		return nil, err
	}
	if !applyProjection {
		return &ObserveCustomerNameResult{Observation: observation, Created: true}, nil
	}
	changed, err := s.applyAutoProjection(ctx, input.CustomerProfileID, preferred, input.SourceEventKey, input.ObservedAt)
	if err != nil {
		return nil, err
	}
	return &ObserveCustomerNameResult{Observation: observation, Created: true, DisplayNameChanged: changed}, nil
}

func (s *CustomerNameObservationService) Pin(ctx context.Context, profileID uint, name, actorRef, eventKey string, at time.Time) error {
	profile, err := s.profiles.FindByID(ctx, profileID)
	if err != nil {
		return fmt.Errorf("pin customer name: find profile: %w", err)
	}
	return s.PinExpected(ctx, profileID, name, profile.RowVersion, actorRef, eventKey, at)
}

func (s *CustomerNameObservationService) PinExpected(ctx context.Context, profileID uint, name string, expectedRowVersion uint64, actorRef, eventKey string, at time.Time) error {
	if strings.TrimSpace(eventKey) == "" {
		return fmt.Errorf("pin customer name: event key is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("pin customer name: name is required")
	}
	if applied, err := s.nameEventAlreadyApplied(ctx, profileID, eventKey, customerNameEventPinned, name); err != nil {
		return err
	} else if applied {
		return nil
	}
	profile, err := s.profiles.FindByID(ctx, profileID)
	if err != nil {
		return fmt.Errorf("pin customer name: find profile: %w", err)
	}
	if err := validateEditableCustomerProfile(profile, expectedRowVersion); err != nil {
		return fmt.Errorf("pin customer name: %w", err)
	}
	observed, err := s.observe(ctx, ObserveCustomerNameInput{
		CustomerProfileID: profileID, Name: name, NameKind: domain.CustomerNameKindManual,
		Authority: actorRef, SourceEventKey: eventKey + ":name", ObservedAt: at,
	}, false)
	if err != nil {
		return err
	}
	if observed.Observation == nil {
		return fmt.Errorf("pin customer name: observation is unavailable")
	}
	if profile.DisplayNameObservationID != nil && *profile.DisplayNameObservationID != observed.Observation.ID {
		if previous, findErr := s.observations.FindByID(ctx, *profile.DisplayNameObservationID); findErr == nil && previous.IsPinned {
			previous.IsPinned = false
			if err := s.observations.Update(ctx, previous); err != nil {
				return fmt.Errorf("pin customer name: clear previous pin: %w", err)
			}
		}
	}
	observed.Observation.IsPinned = true
	observed.Observation.IsActive = true
	if err := s.observations.Update(ctx, observed.Observation); err != nil {
		return fmt.Errorf("pin customer name: update observation: %w", err)
	}
	if s.displayNames == nil {
		return fmt.Errorf("pin customer name: customer profile repository does not support display-name CAS")
	}
	updated, err := s.displayNames.UpdateDisplayNameProjection(ctx, profile.ID, expectedRowVersion, observed.Observation.Name, domain.DisplayNameModePinned, &observed.Observation.ID)
	if err != nil {
		return fmt.Errorf("pin customer name: %w", err)
	}
	if !updated {
		return fmt.Errorf("pin customer name: %w: profile=%d expected=%d", ErrCustomerProfileStale, profileID, expectedRowVersion)
	}
	event := &domain.CustomerNameEvent{
		EventKey: eventKey, CustomerProfileID: profileID, ObservationID: &observed.Observation.ID,
		EventKind: customerNameEventPinned, PreviousName: profile.DisplayName, NewName: name, ReasonCode: "manual_pin",
		ActorRef: actorRef, CreatedAt: normalizedEventTime(at),
	}
	if _, err := s.events.CreateIfAbsent(ctx, event); err != nil {
		return fmt.Errorf("pin customer name: claim event: %w", err)
	}
	return nil
}

func (s *CustomerNameObservationService) Unpin(ctx context.Context, profileID uint, actorRef, eventKey string, at time.Time) error {
	profile, err := s.profiles.FindByID(ctx, profileID)
	if err != nil {
		return fmt.Errorf("unpin customer name: find profile: %w", err)
	}
	return s.UnpinExpected(ctx, profileID, profile.RowVersion, actorRef, eventKey, at)
}

func (s *CustomerNameObservationService) UnpinExpected(ctx context.Context, profileID uint, expectedRowVersion uint64, actorRef, eventKey string, at time.Time) error {
	if err := requireCustomerResolutionFeature(ctx, s.profiles, domain.CustomerResolutionFeatureWrites); err != nil {
		return err
	}
	if strings.TrimSpace(eventKey) == "" {
		return fmt.Errorf("unpin customer name: event key is required")
	}
	if applied, err := s.nameEventAlreadyApplied(ctx, profileID, eventKey, customerNameEventUnpinned, ""); err != nil {
		return err
	} else if applied {
		return nil
	}
	profile, err := s.profiles.FindByID(ctx, profileID)
	if err != nil {
		return fmt.Errorf("unpin customer name: find profile: %w", err)
	}
	if err := validateEditableCustomerProfile(profile, expectedRowVersion); err != nil {
		return fmt.Errorf("unpin customer name: %w", err)
	}
	observations, err := s.observations.ListByProfile(ctx, profileID)
	if err != nil {
		return fmt.Errorf("unpin customer name: list observations: %w", err)
	}
	if profile.DisplayNameObservationID != nil {
		for i := range observations {
			if observations[i].ID == *profile.DisplayNameObservationID {
				observations[i].IsPinned = false
			}
		}
	}
	preferred := preferredObservation(observations)
	name := profile.DisplayName
	var observationID *uint
	if preferred != nil {
		name = preferred.Name
		observationID = &preferred.ID
	}
	if profile.DisplayNameObservationID != nil {
		if pinned, findErr := s.observations.FindByID(ctx, *profile.DisplayNameObservationID); findErr == nil {
			pinned.IsPinned = false
			if err := s.observations.Update(ctx, pinned); err != nil {
				return fmt.Errorf("unpin customer name: clear pin: %w", err)
			}
		}
	}
	if s.displayNames == nil {
		return fmt.Errorf("unpin customer name: customer profile repository does not support display-name CAS")
	}
	updated, err := s.displayNames.UpdateDisplayNameProjection(ctx, profile.ID, expectedRowVersion, name, domain.DisplayNameModeAuto, observationID)
	if err != nil {
		return fmt.Errorf("unpin customer name: %w", err)
	}
	if !updated {
		return fmt.Errorf("unpin customer name: %w: profile=%d expected=%d", ErrCustomerProfileStale, profileID, expectedRowVersion)
	}
	event := &domain.CustomerNameEvent{
		EventKey: eventKey, CustomerProfileID: profileID, ObservationID: observationID,
		EventKind: customerNameEventUnpinned, PreviousName: profile.DisplayName, NewName: name,
		ReasonCode: "manual_unpin", ActorRef: actorRef, CreatedAt: normalizedEventTime(at),
	}
	if _, err := s.events.CreateIfAbsent(ctx, event); err != nil {
		return fmt.Errorf("unpin customer name: claim event: %w", err)
	}
	return nil
}

func (s *CustomerNameObservationService) ListObservations(ctx context.Context, profileID uint) ([]domain.CustomerNameObservation, error) {
	return s.observations.ListByProfile(ctx, profileID)
}

func (s *CustomerNameObservationService) nameEventAlreadyApplied(ctx context.Context, profileID uint, eventKey, eventKind, name string) (bool, error) {
	event, err := s.events.FindByEventKey(ctx, eventKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read customer name idempotency key: %w", err)
	}
	if event.CustomerProfileID != profileID || event.EventKind != eventKind || (name != "" && event.NewName != name) {
		return false, fmt.Errorf("customer name idempotency key %q conflicts with an existing operation", eventKey)
	}
	return true, nil
}

func validateEditableCustomerProfile(profile *domain.CustomerProfile, expectedRowVersion uint64) error {
	if profile.Status == domain.CustomerProfileStatusMerged || profile.MergedIntoProfileID != nil {
		return fmt.Errorf("%w: profile=%d target=%v", ErrCustomerProfileMerged, profile.ID, profile.MergedIntoProfileID)
	}
	if expectedRowVersion == 0 || profile.RowVersion != expectedRowVersion {
		return fmt.Errorf("%w: profile=%d expected=%d actual=%d", ErrCustomerProfileStale, profile.ID, expectedRowVersion, profile.RowVersion)
	}
	return nil
}

func (s *CustomerNameObservationService) rebuildEpisodes(ctx context.Context, profileID, claimedEventID uint) (*domain.CustomerNameObservation, *domain.CustomerNameObservation, error) {
	events, err := s.events.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, nil, fmt.Errorf("observe customer name: list events: %w", err)
	}
	type rawEvent struct {
		event      domain.CustomerNameEvent
		metadata   domain.CustomerNameEventPayload
		normalized string
	}
	raw := make([]rawEvent, 0, len(events))
	for _, event := range events {
		if event.EventKind != customerNameEventObserved {
			continue
		}
		var metadata domain.CustomerNameEventPayload
		if err := json.Unmarshal([]byte(event.Payload), &metadata); err != nil {
			return nil, nil, fmt.Errorf("observe customer name: decode event %q: %w", event.EventKey, err)
		}
		// Early v2 legacy imports stored the raw V1 row as payload. Preserve
		// compatibility with those already-applied data migrations.
		if metadata.NameKind == "" {
			metadata.NameKind = event.ReasonCode
			if metadata.NameKind == "legacy_import" || nameKindPriority(metadata.NameKind) == 0 {
				metadata.NameKind = domain.CustomerNameKindTrustedNickname
			}
			metadata.Authority = event.ActorRef
			metadata.ExtraData = event.Payload
		}
		raw = append(raw, rawEvent{event: event, metadata: metadata, normalized: normalizeCustomerName(event.NewName)})
	}
	sort.SliceStable(raw, func(i, j int) bool {
		if raw[i].event.CreatedAt.Equal(raw[j].event.CreatedAt) {
			return raw[i].event.ID < raw[j].event.ID
		}
		return raw[i].event.CreatedAt.Before(raw[j].event.CreatedAt)
	})
	if err := s.observations.DeactivateByProfile(ctx, profileID); err != nil {
		return nil, nil, fmt.Errorf("observe customer name: deactivate episodes: %w", err)
	}

	var claimed *domain.CustomerNameObservation
	active := make([]domain.CustomerNameObservation, 0, len(raw))
	for start := 0; start < len(raw); {
		end := start + 1
		for end < len(raw) && raw[end].normalized == raw[start].normalized {
			end++
		}
		first := raw[start]
		last := raw[end-1]
		winner := first
		for i := start + 1; i < end; i++ {
			if nameKindPriority(raw[i].metadata.NameKind) >= nameKindPriority(winner.metadata.NameKind) {
				winner = raw[i]
			}
		}
		episode, findErr := s.observations.FindByEpisodeKey(ctx, profileID, first.event.EventKey)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("observe customer name: find episode: %w", findErr)
		}
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			episode, findErr = s.observations.FindBySourceEventKey(ctx, first.event.EventKey)
			if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return nil, nil, fmt.Errorf("observe customer name: find legacy episode: %w", findErr)
			}
		}
		firstSeen, lastSeen := first.event.CreatedAt, last.event.CreatedAt
		if episode == nil {
			episode = &domain.CustomerNameObservation{CustomerProfileID: profileID, EpisodeKey: first.event.EventKey}
		}
		episode.Name = winner.event.NewName
		episode.NormalizedName = winner.normalized
		episode.SourceEventKey = first.event.EventKey
		episode.ObservationCount = uint(end - start)
		episode.NameKind = winner.metadata.NameKind
		episode.Authority = winner.metadata.Authority
		episode.TrustScore = winner.metadata.TrustScore
		episode.SourceIntegrationProfileID = winner.metadata.SourceIntegrationProfileID
		episode.SourceDocumentID = winner.metadata.SourceDocumentID
		episode.SourceIdentityID = winner.metadata.SourceIdentityID
		episode.ObservedAt = &lastSeen
		episode.FirstSeenAt = &firstSeen
		episode.LastSeenAt = &lastSeen
		episode.IsActive = true
		episode.ExtraData = winner.metadata.ExtraData
		if episode.ID == 0 {
			if err := s.observations.Create(ctx, episode); err != nil {
				return nil, nil, fmt.Errorf("observe customer name: create episode: %w", err)
			}
		} else if err := s.observations.Update(ctx, episode); err != nil {
			return nil, nil, fmt.Errorf("observe customer name: update episode: %w", err)
		}
		for i := start; i < end; i++ {
			if err := s.events.UpdateObservationID(ctx, raw[i].event.ID, episode.ID); err != nil {
				return nil, nil, fmt.Errorf("observe customer name: link event to episode: %w", err)
			}
			if raw[i].event.ID == claimedEventID {
				copy := *episode
				claimed = &copy
			}
		}
		active = append(active, *episode)
		start = end
	}
	return claimed, preferredObservation(active), nil
}

func (s *CustomerNameObservationService) applyAutoProjection(ctx context.Context, profileID uint, preferred *domain.CustomerNameObservation, eventKey string, at time.Time) (bool, error) {
	if preferred == nil {
		return false, nil
	}
	profile, err := s.profiles.FindByID(ctx, profileID)
	if err != nil {
		return false, fmt.Errorf("observe customer name: find profile: %w", err)
	}
	if profile.DisplayNameMode == domain.DisplayNameModePinned {
		return false, nil
	}
	if profile.DisplayName == preferred.Name && profile.DisplayNameObservationID != nil && *profile.DisplayNameObservationID == preferred.ID {
		return false, nil
	}
	previous := profile.DisplayName
	if err := s.updateProjection(ctx, profile, preferred.Name, domain.DisplayNameModeAuto, &preferred.ID); err != nil {
		return false, fmt.Errorf("observe customer name: %w", err)
	}
	projectionEvent := &domain.CustomerNameEvent{
		EventKey: "projection:" + eventKey, CustomerProfileID: profileID, ObservationID: &preferred.ID,
		EventKind: customerNameEventProjectionChanged, PreviousName: previous, NewName: preferred.Name,
		ReasonCode: "preferred_observation", CreatedAt: normalizedEventTime(at),
	}
	if _, err := s.events.CreateIfAbsent(ctx, projectionEvent); err != nil {
		return false, fmt.Errorf("observe customer name: record projection event: %w", err)
	}
	return previous != preferred.Name, nil
}

func (s *CustomerNameObservationService) updateProjection(ctx context.Context, profile *domain.CustomerProfile, name, mode string, observationID *uint) error {
	if s.displayNames == nil {
		return fmt.Errorf("customer profile repository does not support display-name CAS")
	}
	for attempts := 0; attempts < 3; attempts++ {
		updated, err := s.displayNames.UpdateDisplayNameProjection(ctx, profile.ID, profile.RowVersion, name, mode, observationID)
		if err != nil {
			return err
		}
		if updated {
			return nil
		}
		profile, err = s.profiles.FindByID(ctx, profile.ID)
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("display-name projection changed concurrently")
}

func preferredObservation(observations []domain.CustomerNameObservation) *domain.CustomerNameObservation {
	var preferred *domain.CustomerNameObservation
	for i := range observations {
		candidate := &observations[i]
		priority := nameKindPriority(candidate.NameKind)
		if !candidate.IsActive || candidate.IsPinned || priority <= 0 {
			continue
		}
		if preferred == nil || priority > nameKindPriority(preferred.NameKind) ||
			(priority == nameKindPriority(preferred.NameKind) && observationAfter(candidate, preferred)) {
			copy := *candidate
			preferred = &copy
		}
	}
	return preferred
}

func observationAfter(left, right *domain.CustomerNameObservation) bool {
	leftAt, rightAt := left.LastSeenAt, right.LastSeenAt
	if leftAt == nil {
		return false
	}
	if rightAt == nil {
		return true
	}
	if leftAt.Equal(*rightAt) {
		return left.ID > right.ID
	}
	return leftAt.After(*rightAt)
}

func nameKindPriority(kind string) int {
	switch kind {
	case domain.CustomerNameKindStableIdentityNickname:
		return 300
	case domain.CustomerNameKindTrustedNickname:
		return 200
	case domain.CustomerNameKindRecipient:
		return 100
	default:
		return 0
	}
}

func normalizeCustomerName(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(value))
}

func normalizedEventTime(at time.Time) time.Time {
	if at.IsZero() {
		return time.Now().UTC()
	}
	return at.UTC()
}
