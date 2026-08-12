package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

const IdentityStrategyOrderScopedProvisional = "order_scoped_provisional"

type AmbiguousIdentityError struct {
	Namespace       string
	IdentityType    string
	NormalizedValue string
	Count           int
}

func (e *AmbiguousIdentityError) Error() string {
	return fmt.Sprintf("ambiguous customer identity %s/%s/%s: %d active rows", e.Namespace, e.IdentityType, e.NormalizedValue, e.Count)
}

type StableIdentityResolutionInput struct {
	Namespace                  string
	IdentityPlatform           string
	IdentityValue              string
	IdentityType               string
	SourceIntegrationProfileID *uint
	ObservedAt                 time.Time
	InitialDisplayName         string
	ProfileType                string
}

type StableIdentityResolutionResult struct {
	CustomerProfileID  uint
	CustomerIdentityID uint
	Created            bool
}

// IdentityResolutionService resolves canonical strong identities. The service
// deliberately returns ambiguity instead of picking the first legacy duplicate.
type IdentityResolutionService struct {
	profileRepo domain.CustomerProfileRepository
	strongRepo  domain.StrongIdentityRepository
}

func NewIdentityResolutionService(profileRepo domain.CustomerProfileRepository) *IdentityResolutionService {
	strongRepo, _ := profileRepo.(domain.StrongIdentityRepository)
	return &IdentityResolutionService{profileRepo: profileRepo, strongRepo: strongRepo}
}

// ResolveStableProfile must be called with a transaction-bound repository when
// the caller requires profile+identity creation to be atomic.
func (s *IdentityResolutionService) ResolveStableProfile(ctx context.Context, input StableIdentityResolutionInput) (*StableIdentityResolutionResult, error) {
	if err := requireCustomerResolutionFeature(ctx, s.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	namespace := strings.ToLower(strings.TrimSpace(input.Namespace))
	if namespace == "" {
		namespace = strings.ToLower(strings.TrimSpace(input.IdentityPlatform))
	}
	identityType := strings.TrimSpace(input.IdentityType)
	value := strings.TrimSpace(input.IdentityValue)
	if namespace == "" || value == "" || identityType == "" {
		return nil, fmt.Errorf("identity namespace, value, and type are required for resolution")
	}
	switch identityType {
	case string(domain.IdentityTypePlatformUID), string(domain.IdentityTypeEmail), string(domain.IdentityTypeExternalBuyerID):
	default:
		return nil, fmt.Errorf("identity type %q is not a stable resolver key", identityType)
	}
	normalizedValue := normalizeIdentityValue(identityType, value)
	at := input.ObservedAt.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}

	var identities []domain.CustomerIdentity
	var err error
	if s.strongRepo != nil {
		identities, err = s.strongRepo.ListByResolutionKey(ctx, namespace, identityType, normalizedValue)
	} else {
		legacy, findErr := s.profileRepo.FindIdentityByPlatformAndValue(ctx, input.IdentityPlatform, value)
		switch {
		case findErr == nil && legacy != nil:
			identities = []domain.CustomerIdentity{*legacy}
		case errors.Is(findErr, gorm.ErrRecordNotFound):
		case findErr != nil:
			err = findErr
		}
	}
	if err != nil {
		return nil, fmt.Errorf("find identity by canonical key: %w", err)
	}
	if len(identities) > 1 {
		return nil, &AmbiguousIdentityError{Namespace: namespace, IdentityType: identityType, NormalizedValue: normalizedValue, Count: len(identities)}
	}
	if len(identities) == 1 {
		identity := identities[0]
		identity.Namespace = namespace
		identity.NormalizedValue = normalizedValue
		identity.NormalizationVersion = "v1"
		if identity.Authority == "" {
			identity.Authority = "source_platform"
		}
		if identity.VerificationStatus == "" || identity.VerificationStatus == "unverified" {
			identity.VerificationStatus = "observed"
		}
		identity.ResolutionStatus = "resolved"
		if identity.FirstSeenAt == nil {
			identity.FirstSeenAt = &at
		}
		identity.LastSeenAt = &at
		if identity.SourceIntegrationProfileID == nil {
			identity.SourceIntegrationProfileID = input.SourceIntegrationProfileID
		}
		if s.strongRepo != nil {
			if err := s.strongRepo.UpdateResolutionMetadata(ctx, &identity); err != nil {
				return nil, fmt.Errorf("update identity resolution metadata: %w", err)
			}
		}
		return &StableIdentityResolutionResult{CustomerProfileID: identity.CustomerProfileID, CustomerIdentityID: identity.ID}, nil
	}

	displayName := strings.TrimSpace(input.InitialDisplayName)
	if displayName == "" {
		displayName = value
	}
	profileType := strings.TrimSpace(input.ProfileType)
	if profileType == "" {
		profileType = "manual"
	}
	profile := &domain.CustomerProfile{
		DisplayName: displayName, ProfileType: profileType, Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto, CreatedAt: at, UpdatedAt: at,
	}
	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile for identity: %w", err)
	}
	platform := strings.TrimSpace(input.IdentityPlatform)
	if platform == "" {
		platform = namespace
	}
	identity := &domain.CustomerIdentity{
		CustomerProfileID: profile.ID, IdentityPlatform: platform, IdentityValue: value,
		IdentityType: identityType, Namespace: namespace, NormalizedValue: normalizedValue,
		NormalizationVersion: "v1", Authority: "source_platform", VerificationStatus: "observed",
		SourceIntegrationProfileID: input.SourceIntegrationProfileID, ResolutionStatus: "resolved",
		FirstSeenAt: &at, LastSeenAt: &at, IsPrimary: true, CreatedAt: at, UpdatedAt: at,
	}
	if err := s.profileRepo.CreateIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("create identity: %w", err)
	}
	return &StableIdentityResolutionResult{CustomerProfileID: profile.ID, CustomerIdentityID: identity.ID, Created: true}, nil
}

// ResolveOrCreateProfile is retained for legacy call sites; new demand imports
// use ResolveStableProfile inside their outer transaction.
func (s *IdentityResolutionService) ResolveOrCreateProfile(ctx context.Context, platform, identityValue, identityType string) (uint, error) {
	result, err := s.ResolveStableProfile(ctx, StableIdentityResolutionInput{
		Namespace: platform, IdentityPlatform: platform, IdentityValue: identityValue,
		IdentityType: identityType, ObservedAt: time.Now().UTC(),
	})
	if err != nil {
		return 0, err
	}
	return result.CustomerProfileID, nil
}

func normalizeIdentityValue(identityType, value string) string {
	value = strings.TrimSpace(value)
	if identityType == "email" {
		return strings.ToLower(value)
	}
	return value
}

// ResolveIdentityStrategy maps stable strategies to CustomerIdentity types.
// order_scoped_provisional is intentionally not a CustomerIdentity type.
func ResolveIdentityStrategy(strategy string) string {
	switch strategy {
	case "email":
		return "email"
	case "external_buyer_id":
		return "external_buyer_id"
	case IdentityStrategyOrderScopedProvisional:
		return ""
	default:
		return "platform_uid"
	}
}
