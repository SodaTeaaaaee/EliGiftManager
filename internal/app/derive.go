package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type ResultView struct {
	Result          domain.FulfillmentResult
	WorkState       domain.WorkState
	Blocks          []domain.BlockReason
	InFactory       bool
	Shipped         bool
	WritebackFailed bool
}

func (ws *Workspace) inspectResult(ctx context.Context, r domain.FulfillmentResult) (ResultView, error) {
	view := ResultView{Result: r}
	if r.ProductItemID == nil {
		view.Blocks = append(view.Blocks, domain.BlockUnalignedProduct)
	}
	if !r.Address.Usable() {
		view.Blocks = append(view.Blocks, domain.BlockUnusableAddress)
	}
	if r.SourceKind == string(domain.SourceEntitlementInstance) && r.EntitlementInstanceID != nil {
		inst, err := ws.Store.GetInstance(ctx, *r.EntitlementInstanceID)
		if err != nil {
			return view, err
		}
		if inst.PlatformIdentityID != nil {
			ident, err := ws.Store.GetIdentity(ctx, *inst.PlatformIdentityID)
			if err != nil {
				return view, err
			}
			if ident.CustomerProfileID == nil {
				view.Blocks = append(view.Blocks, domain.BlockIdentityUnattached)
			}
		} else {
			view.Blocks = append(view.Blocks, domain.BlockIdentityUnattached)
		}
	}
	links, err := ws.Store.ListLinksByResult(ctx, r.ID)
	if err != nil {
		return view, err
	}
	view.InFactory = r.Frozen && len(links) > 0
	for _, link := range links {
		line, err := ws.Store.GetSupplierOrderLine(ctx, link.SupplierOrderLineID)
		if err != nil {
			return view, err
		}
		if line.TrackingRetired {
			continue
		}
		ships, err := ws.Store.ListShipmentsByTracking(ctx, line.TrackingID)
		if err != nil {
			return view, err
		}
		if len(ships) > 0 {
			view.Shipped = true
		}
	}
	if r.InputFactID != nil {
		wbs, err := ws.Store.ListWritebacksByFact(ctx, *r.InputFactID)
		if err != nil {
			return view, err
		}
		for _, wb := range wbs {
			if wb.Status == string(domain.WritebackFailed) {
				view.WritebackFailed = true
				break
			}
		}
	}
	switch {
	case len(view.Blocks) > 0:
		view.WorkState = domain.WorkStateBlocked
	case view.WritebackFailed:
		view.WorkState = domain.WorkStateWritebackFailed
	case view.Shipped:
		view.WorkState = domain.WorkStateShipped
	case view.InFactory:
		view.WorkState = domain.WorkStateInFactory
	default:
		view.WorkState = domain.WorkStateReady
	}
	return view, nil
}

// defaultSnapshot derives the address snapshot a new result starts from: the
// customer's default address, or the first one when no default is flagged.
// Every read goes through the explicit store argument so callers control
// which transaction or connection the lookup joins.
func defaultSnapshot(ctx context.Context, store domain.Store, customerID *uint) (domain.AddressSnapshot, error) {
	if customerID == nil {
		return domain.AddressSnapshot{}, nil
	}
	addrs, err := store.ListAddresses(ctx, *customerID)
	if err != nil {
		return domain.AddressSnapshot{}, err
	}
	var chosen *domain.RecipientAddress
	for i := range addrs {
		if addrs[i].IsDefault {
			chosen = &addrs[i]
			break
		}
	}
	if chosen == nil && len(addrs) > 0 {
		chosen = &addrs[0]
	}
	if chosen == nil {
		return domain.AddressSnapshot{}, nil
	}
	return snapshotFromAddress(*chosen), nil
}

func snapshotFromAddress(a domain.RecipientAddress) domain.AddressSnapshot {
	id := a.ID
	return domain.AddressSnapshot{
		SourceAddressID: &id,
		RecipientName:   a.RecipientName,
		Phone:           a.Phone,
		Country:         a.Country,
		Province:        a.Province,
		City:            a.City,
		District:        a.District,
		AddressLine1:    a.AddressLine1,
		AddressLine2:    a.AddressLine2,
		PostalCode:      a.PostalCode,
	}
}
