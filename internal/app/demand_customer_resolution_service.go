package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// DemandCustomerResolutionInput contains the profile-controlled identity
// settings and the document facts needed to resolve a demand's customer.
type DemandCustomerResolutionInput struct {
	IntegrationProfileID uint
	IdentityStrategy     string
	SourceChannel        string
	SourceDocumentNo     string
	SourceCustomerRef    string
	DisplayName          string
	ObservedAt           time.Time
}

// DemandCustomerResolutionResult identifies the rows created or reused by a
// demand import. IdentityID is set only for stable membership resolution;
// OriginID is set only for order-scoped provisional resolution.
type DemandCustomerResolutionResult struct {
	CustomerProfileID *uint
	IdentityID        *uint
	OriginID          *uint
}

// DemandCustomerResolutionService is the single strategy dispatcher used by
// all demand intake entry points. Its repositories must be bound to the same
// transaction that persists the demand document.
type DemandCustomerResolutionService struct {
	identities *IdentityResolutionService
	retail     *CustomerResolutionService
	origins    domain.CustomerProfileOriginRepository
}

func NewDemandCustomerResolutionService(
	profiles domain.CustomerProfileRepository,
	origins domain.CustomerProfileOriginRepository,
) *DemandCustomerResolutionService {
	return &DemandCustomerResolutionService{
		identities: NewIdentityResolutionService(profiles),
		retail:     NewCustomerResolutionService(profiles, origins),
		origins:    origins,
	}
}

// Resolve dispatches by the integration profile's identity strategy. Buyer
// nicknames/sourceCustomerRef never participate in stable identity resolution
// for an order_scoped_provisional profile; the stable key is the order origin
// (integrationProfileID, sourceDocumentNo).
func (s *DemandCustomerResolutionService) Resolve(
	ctx context.Context,
	input DemandCustomerResolutionInput,
) (*DemandCustomerResolutionResult, error) {
	at := input.ObservedAt.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}

	if input.IdentityStrategy == IdentityStrategyOrderScopedProvisional {
		if input.IntegrationProfileID == 0 {
			return nil, fmt.Errorf("order_scoped_provisional resolution requires integrationProfileId")
		}
		if strings.TrimSpace(input.SourceDocumentNo) == "" {
			return nil, fmt.Errorf("order_scoped_provisional resolution requires sourceDocumentNo")
		}
		resolved, err := s.retail.ResolveRetailOrderProfile(
			ctx,
			input.IntegrationProfileID,
			input.SourceDocumentNo,
			input.DisplayName,
			at,
		)
		if err != nil {
			return nil, err
		}
		profileID, originID := resolved.CustomerProfileID, resolved.OriginID
		return &DemandCustomerResolutionResult{CustomerProfileID: &profileID, OriginID: &originID}, nil
	}

	identityValue := strings.TrimSpace(input.SourceCustomerRef)
	namespace := strings.TrimSpace(input.SourceChannel)
	if identityValue == "" || namespace == "" {
		return nil, fmt.Errorf("stable customer resolution requires sourceCustomerRef and sourceChannel")
	}
	identityType := ResolveIdentityStrategy(input.IdentityStrategy)
	if identityType == "" {
		return nil, fmt.Errorf("stable customer resolution cannot use identity strategy %q", input.IdentityStrategy)
	}
	profileID := input.IntegrationProfileID
	resolved, err := s.identities.ResolveStableProfile(ctx, StableIdentityResolutionInput{
		Namespace:                  namespace,
		IdentityPlatform:           namespace,
		IdentityValue:              identityValue,
		IdentityType:               identityType,
		SourceIntegrationProfileID: &profileID,
		ObservedAt:                 at,
		InitialDisplayName:         input.DisplayName,
		ProfileType:                "member",
	})
	if err != nil {
		return nil, err
	}
	resolvedProfileID, identityID := resolved.CustomerProfileID, resolved.CustomerIdentityID
	return &DemandCustomerResolutionResult{CustomerProfileID: &resolvedProfileID, IdentityID: &identityID}, nil
}

// AttachOriginDocument records the first demand document associated with an
// order-scoped origin. Retries keep the original source-document link.
func (s *DemandCustomerResolutionService) AttachOriginDocument(ctx context.Context, originID, documentID uint) error {
	origin, err := s.origins.FindByID(ctx, originID)
	if err != nil {
		return fmt.Errorf("load customer origin %d: %w", originID, err)
	}
	if origin.SourceDocumentID != nil {
		return nil
	}
	origin.SourceDocumentID = &documentID
	if err := s.origins.Update(ctx, origin); err != nil {
		return fmt.Errorf("attach demand document to customer origin: %w", err)
	}
	return nil
}
