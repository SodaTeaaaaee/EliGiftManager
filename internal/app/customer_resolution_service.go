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

type RetailOrderResolutionResult struct {
	CustomerProfileID uint
	OriginID          uint
	Created           bool
}

type CustomerResolutionService struct {
	profiles domain.CustomerProfileRepository
	origins  domain.CustomerProfileOriginRepository
}

func NewCustomerResolutionService(profiles domain.CustomerProfileRepository, origins domain.CustomerProfileOriginRepository) *CustomerResolutionService {
	return &CustomerResolutionService{profiles: profiles, origins: origins}
}

// ResolveRetailOrderProfile creates at most one provisional profile per
// (integration profile, order number). Buyer nicknames are display facts only.
func (s *CustomerResolutionService) ResolveRetailOrderProfile(
	ctx context.Context,
	integrationProfileID uint,
	orderNo, displayName string,
	observedAt time.Time,
) (*RetailOrderResolutionResult, error) {
	if err := requireCustomerResolutionFeature(ctx, s.profiles, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	orderNo = strings.TrimSpace(orderNo)
	if integrationProfileID == 0 || orderNo == "" {
		return nil, fmt.Errorf("retail order resolution requires integration profile and order number")
	}
	at := observedAt.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	existing, err := s.origins.FindByExternalRef(ctx, domain.CustomerOriginKindRetailOrder, integrationProfileID, orderNo)
	if err == nil && existing != nil {
		if existing.LastSeenAt == nil || at.After(*existing.LastSeenAt) {
			existing.LastSeenAt = &at
		}
		if err := s.origins.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("update retail order origin: %w", err)
		}
		return &RetailOrderResolutionResult{CustomerProfileID: existing.CustomerProfileID, OriginID: existing.ID}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("find retail order origin: %w", err)
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = orderNo
	}
	profile := &domain.CustomerProfile{
		DisplayName: displayName, ProfileType: "buyer", Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto, CreatedAt: at, UpdatedAt: at,
	}
	if err := s.profiles.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("create provisional retail profile: %w", err)
	}
	origin := &domain.CustomerProfileOrigin{
		CustomerProfileID: profile.ID, OriginKind: domain.CustomerOriginKindRetailOrder,
		SourceIntegrationProfileID: &integrationProfileID, ExternalRef: orderNo, IsProvisional: true,
		FirstSeenAt: &at, LastSeenAt: &at, CreatedAt: at, UpdatedAt: at,
	}
	created, err := s.origins.CreateIfAbsent(ctx, origin)
	if err != nil {
		return nil, fmt.Errorf("create retail order origin: %w", err)
	}
	if !created && origin.CustomerProfileID != profile.ID {
		return nil, fmt.Errorf("retail order origin was created concurrently; retry transaction")
	}
	return &RetailOrderResolutionResult{CustomerProfileID: profile.ID, OriginID: origin.ID, Created: true}, nil
}
