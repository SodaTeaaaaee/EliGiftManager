package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

// IdentityResolutionService resolves or creates CustomerProfiles based on identity lookup.
type IdentityResolutionService struct {
	profileRepo domain.CustomerProfileRepository
}

func NewIdentityResolutionService(profileRepo domain.CustomerProfileRepository) *IdentityResolutionService {
	return &IdentityResolutionService{profileRepo: profileRepo}
}

// ResolveOrCreateProfile looks up an existing CustomerIdentity by platform + value.
// If found, returns the associated CustomerProfileID.
// If not found, creates a new CustomerProfile + CustomerIdentity and returns the new ID.
func (s *IdentityResolutionService) ResolveOrCreateProfile(ctx context.Context, platform, identityValue, identityType string) (uint, error) {
	if platform == "" || identityValue == "" {
		return 0, fmt.Errorf("identity platform and value are required for resolution")
	}
	existing, err := s.profileRepo.FindIdentityByPlatformAndValue(ctx, platform, identityValue)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("find identity by platform and value: %w", err)
		}
	} else if existing != nil {
		return existing.CustomerProfileID, nil
	}

	// Create new profile + identity
	now := time.Now()
	profile := &domain.CustomerProfile{
		DisplayName: identityValue,
		ProfileType: "manual",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return 0, fmt.Errorf("create profile for identity: %w", err)
	}

	identity := &domain.CustomerIdentity{
		CustomerProfileID: profile.ID,
		IdentityPlatform:  platform,
		IdentityValue:     identityValue,
		IdentityType:      identityType,
		IsPrimary:         true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.profileRepo.CreateIdentity(ctx, identity); err != nil {
		return 0, fmt.Errorf("create identity: %w", err)
	}

	return profile.ID, nil
}

// ResolveIdentityStrategy maps an IntegrationProfile.IdentityStrategy to the corresponding IdentityType.
func ResolveIdentityStrategy(strategy string) string {
	switch strategy {
	case "email":
		return "email"
	case "external_buyer_id":
		return "external_buyer_id"
	default:
		return "platform_uid"
	}
}
