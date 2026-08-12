package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func identityResolutionKey(identity domain.CustomerIdentity) string {
	namespace := strings.ToLower(strings.TrimSpace(identity.Namespace))
	if namespace == "" {
		namespace = strings.ToLower(strings.TrimSpace(identity.IdentityPlatform))
	}
	return namespace + "\x00" + strings.ToLower(strings.TrimSpace(identity.IdentityType))
}

func identityResolutionValue(identity domain.CustomerIdentity) string {
	value := strings.TrimSpace(identity.NormalizedValue)
	if value == "" {
		value = strings.TrimSpace(identity.IdentityValue)
	}
	if strings.EqualFold(identity.IdentityType, string(domain.IdentityTypeEmail)) {
		value = strings.ToLower(value)
	}
	return value
}

func isStrongMergeIdentity(identity domain.CustomerIdentity) bool {
	return strings.EqualFold(identity.IdentityType, string(domain.IdentityTypePlatformUID)) ||
		strings.EqualFold(identity.IdentityType, string(domain.IdentityTypeExternalBuyerID))
}

func chooseIdentityPrimary(source, target []domain.CustomerIdentity, selections []dto.PrimaryIdentitySelection) (map[uint]bool, []dto.MergeBlocker) {
	all := append(append([]domain.CustomerIdentity{}, source...), target...)
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	groups := map[string][]domain.CustomerIdentity{}
	for _, identity := range all {
		groups[identityResolutionKey(identity)] = append(groups[identityResolutionKey(identity)], identity)
	}
	selectionByGroup := map[string]uint{}
	blockers := make([]dto.MergeBlocker, 0)
	for _, selection := range selections {
		key := strings.ToLower(strings.TrimSpace(selection.Namespace)) + "\x00" + strings.ToLower(strings.TrimSpace(selection.IdentityType))
		if previous, ok := selectionByGroup[key]; ok && previous != selection.IdentityID {
			blockers = append(blockers, mergeBlocker("multiple_primary_selections", domain.MergeEntityIdentity, selection.IdentityID, key))
		}
		selectionByGroup[key] = selection.IdentityID
	}
	result := map[uint]bool{}
	for key, identities := range groups {
		sourcePrimaries, targetPrimaries := make([]uint, 0), make([]uint, 0)
		for _, identity := range identities {
			if !identity.IsPrimary {
				continue
			}
			if containsIdentity(source, identity.ID) {
				sourcePrimaries = append(sourcePrimaries, identity.ID)
			} else {
				targetPrimaries = append(targetPrimaries, identity.ID)
			}
		}
		if len(sourcePrimaries) > 1 || len(targetPrimaries) > 1 {
			blockers = append(blockers, mergeBlocker("invalid_multiple_primary", domain.MergeEntityIdentity, 0, key))
		}
		winner := uint(0)
		if selected := selectionByGroup[key]; selected != 0 {
			for _, identity := range identities {
				if identity.ID == selected {
					winner = selected
					break
				}
			}
			if winner == 0 {
				blockers = append(blockers, mergeBlocker("invalid_primary_selection", domain.MergeEntityIdentity, selected, key))
			}
		} else if len(targetPrimaries) == 1 {
			winner = targetPrimaries[0]
		} else if len(sourcePrimaries) == 1 {
			winner = sourcePrimaries[0]
		}
		for _, identity := range identities {
			result[identity.ID] = winner != 0 && identity.ID == winner
		}
	}
	for key, selected := range selectionByGroup {
		if _, ok := groups[key]; !ok {
			blockers = append(blockers, mergeBlocker("unknown_primary_group", domain.MergeEntityIdentity, selected, key))
		}
	}
	return result, blockers
}

func stableIdentityBlockers(source, target []domain.CustomerIdentity) []dto.MergeBlocker {
	left, right := map[string]map[string]struct{}{}, map[string]map[string]struct{}{}
	collect := func(rows []domain.CustomerIdentity, into map[string]map[string]struct{}) {
		for _, identity := range rows {
			if !isStrongMergeIdentity(identity) {
				continue
			}
			key, value := identityResolutionKey(identity), identityResolutionValue(identity)
			if value == "" {
				continue
			}
			if into[key] == nil {
				into[key] = map[string]struct{}{}
			}
			into[key][value] = struct{}{}
		}
	}
	collect(source, left)
	collect(target, right)
	blockers := make([]dto.MergeBlocker, 0)
	for key, leftValues := range left {
		rightValues := right[key]
		if len(rightValues) == 0 || sameStringSet(leftValues, rightValues) {
			continue
		}
		blockers = append(blockers, mergeBlocker("strong_identity_conflict", domain.MergeEntityIdentity, 0, key))
	}
	return blockers
}

func sameStringSet(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, ok := right[value]; !ok {
			return false
		}
	}
	return true
}

func chooseDefaultAddress(source, target []domain.CustomerAddress, requested *uint) (map[uint]bool, []dto.MergeBlocker) {
	all := append(append([]domain.CustomerAddress{}, source...), target...)
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	sourceDefaults, targetDefaults := make([]uint, 0), make([]uint, 0)
	for _, address := range source {
		if address.IsDefault {
			sourceDefaults = append(sourceDefaults, address.ID)
		}
	}
	for _, address := range target {
		if address.IsDefault {
			targetDefaults = append(targetDefaults, address.ID)
		}
	}
	blockers := make([]dto.MergeBlocker, 0)
	if len(sourceDefaults) > 1 || len(targetDefaults) > 1 {
		blockers = append(blockers, mergeBlocker("invalid_multiple_default_addresses", domain.MergeEntityAddress, 0, ""))
	}
	winner := uint(0)
	if requested != nil {
		for _, address := range all {
			if address.ID == *requested {
				winner = address.ID
				break
			}
		}
		if winner == 0 {
			blockers = append(blockers, mergeBlocker("invalid_default_address_selection", domain.MergeEntityAddress, *requested, ""))
		}
	} else if len(targetDefaults) == 1 {
		winner = targetDefaults[0]
	} else if len(sourceDefaults) == 1 {
		winner = sourceDefaults[0]
	}
	result := map[uint]bool{}
	for _, address := range all {
		result[address.ID] = winner != 0 && address.ID == winner
	}
	return result, blockers
}

func buildPrimaryIdentityOptions(source, target []domain.CustomerIdentity, chosen map[uint]bool) ([]dto.PrimaryIdentityOption, []dto.PrimaryIdentitySelection) {
	all := append(append([]domain.CustomerIdentity{}, target...), source...)
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	options := make([]dto.PrimaryIdentityOption, 0, len(all))
	recommended := make([]dto.PrimaryIdentitySelection, 0)
	for _, identity := range all {
		namespace := strings.TrimSpace(identity.Namespace)
		if namespace == "" {
			namespace = strings.TrimSpace(identity.IdentityPlatform)
		}
		options = append(options, dto.PrimaryIdentityOption{Namespace: namespace, IdentityType: string(identity.IdentityType), IdentityID: identity.ID, CustomerProfileID: identity.CustomerProfileID, DisplayValue: identity.IdentityValue, CurrentPrimary: identity.IsPrimary})
		if chosen[identity.ID] {
			recommended = append(recommended, dto.PrimaryIdentitySelection{Namespace: namespace, IdentityType: string(identity.IdentityType), IdentityID: identity.ID})
		}
	}
	return options, recommended
}

func buildDefaultAddressOptions(source, target []domain.CustomerAddress, chosen map[uint]bool) ([]dto.DefaultAddressOption, *uint) {
	all := append(append([]domain.CustomerAddress{}, target...), source...)
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	options := make([]dto.DefaultAddressOption, 0, len(all))
	var recommended *uint
	for _, address := range all {
		display := strings.TrimSpace(strings.Join([]string{address.RecipientName, address.Province, address.City, address.AddressLine1}, " "))
		options = append(options, dto.DefaultAddressOption{AddressID: address.ID, CustomerProfileID: address.CustomerProfileID, DisplayValue: display, CurrentDefault: address.IsDefault})
		if chosen[address.ID] {
			id := address.ID
			recommended = &id
		}
	}
	return options, recommended
}

func buildDisplayNameOptions(source, target *domain.CustomerProfile) ([]dto.DisplayNameOption, string) {
	options := []dto.DisplayNameOption{{Resolution: "keep_target", DisplayName: target.DisplayName, ProfileID: target.ID}, {Resolution: "keep_source", DisplayName: source.DisplayName, ProfileID: source.ID}}
	recommended := "keep_target"
	if target.DisplayNameMode != domain.DisplayNameModePinned && source.DisplayNameMode == domain.DisplayNameModePinned {
		recommended = "keep_source"
	}
	return options, recommended
}

func containsIdentity(rows []domain.CustomerIdentity, id uint) bool {
	for _, row := range rows {
		if row.ID == id {
			return true
		}
	}
	return false
}

func mergeBlocker(code, entityType string, entityID uint, detail string) dto.MergeBlocker {
	return dto.MergeBlocker{Code: code, EntityType: entityType, EntityID: entityID, Detail: detail}
}

func originMergeKey(origin domain.CustomerProfileOrigin) string {
	integration := "nil"
	if origin.SourceIntegrationProfileID != nil {
		integration = fmt.Sprint(*origin.SourceIntegrationProfileID)
	}
	return strings.ToLower(strings.TrimSpace(origin.OriginKind)) + "\x00" + integration + "\x00" + strings.TrimSpace(origin.ExternalRef)
}
