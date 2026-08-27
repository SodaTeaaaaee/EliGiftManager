package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type IngestLine struct {
	SourceLineNo  int
	ExternalSKU   string
	ExternalTitle string
	ExternalSpec  string
	Quantity      int
}

type IngestFactInput struct {
	Kind              string
	StableExternalID  string
	IdentityType      string
	IdentityValue     string
	MembershipLevel   string
	SourceDocumentNo  string
	SourceCreatedAt   *time.Time
	CustomerProfileID *uint
	ExtraData         string
	Lines             []IngestLine
}

type InboxRow struct {
	Line       domain.InputFactLine
	Fact       domain.InputFact
	Document   *domain.InputDocument
	Assigned   bool
	Unaligned  bool
	Unattached bool
	// RevisionPending marks rows of a revision fact awaiting an apply or
	// dismiss decision; they are visible but not assignable.
	RevisionPending bool
	// AliasID resolves the line's external SKU to its product alias, when
	// one exists, so alignment actions can target it.
	AliasID *uint
}

type IngestDocumentResult struct {
	Document   domain.InputDocument
	Duplicates []domain.DuplicateObservation
}

func (ws *Workspace) IngestDocument(ctx context.Context, doc *domain.InputDocument, facts []IngestFactInput) (*domain.InputDocument, []domain.DuplicateObservation, error) {
	var (
		outDoc *domain.InputDocument
		dups   []domain.DuplicateObservation
	)
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		d, obs, err := ws.withStore(tx).ingestDocument(ctx, tx, doc, facts)
		if err != nil {
			return err
		}
		outDoc, dups = d, obs
		return nil
	}); err != nil {
		return nil, nil, err
	}
	return outDoc, dups, nil
}

// ingestDocument creates the document together with its identities, facts,
// fact lines, and duplicate observations. Every read and write goes through
// the explicit store argument, so the whole import commits or rolls back as
// one unit.
func (ws *Workspace) ingestDocument(ctx context.Context, tx domain.Store, doc *domain.InputDocument, facts []IngestFactInput) (*domain.InputDocument, []domain.DuplicateObservation, error) {
	settings, err := tx.GetSettings(ctx)
	if err != nil {
		return nil, nil, err
	}
	doc.ImportedAt = ws.Now()
	if doc.Direction == "" {
		doc.Direction = string(domain.TemplateDirectionInput)
	}
	if err := tx.CreateDocument(ctx, doc); err != nil {
		return nil, nil, err
	}
	var dups []domain.DuplicateObservation
	for _, in := range facts {
		obs, err := ws.ingestOneFact(ctx, tx, doc, in, settings)
		if err != nil {
			return nil, nil, err
		}
		if obs != nil {
			dups = append(dups, *obs)
		}
	}
	return doc, dups, nil
}

func (ws *Workspace) ingestOneFact(ctx context.Context, tx domain.Store, doc *domain.InputDocument, in IngestFactInput, settings *domain.AppSettings) (*domain.DuplicateObservation, error) {
	kind := in.Kind
	if kind == "" {
		kind = string(domain.InputFactKindRetailOrder)
	}
	if in.StableExternalID != "" {
		existing, err := tx.FindFactByStableID(ctx, doc.PlatformID, in.StableExternalID)
		if err == nil {
			// Same stable id: identical content is a duplicate, different
			// content is a revision of the established fact.
			inFps := in.ingestFingerprints()
			same, err := sameContentFingerprints(ctx, tx, *existing, inFps)
			if err != nil {
				return nil, err
			}
			exFps, _ := factLineFingerprints(ctx, tx, *existing)
			println("DEBUG verdict", existing.ID, same)
			for _, f := range exFps {
				println("  ex ", f)
			}
			for _, f := range inFps {
				println("  in ", f)
			}
			if !same {
				if _, err := ws.createFactWithLines(ctx, tx, doc, in, kind, &existing.ID); err != nil {
					return nil, err
				}
				return nil, nil
			}
			return ws.recordObservation(ctx, tx, doc, in, existing.ID, domain.DuplicateRecordOnly, "stable_external_id", true)
		}
		if err != domain.ErrNotFound {
			return nil, err
		}
	} else if in.SourceCreatedAt != nil {
		// No stable external id: judge by content fingerprint against existing
		// facts of the same platform and kind, then by the interval since the
		// matched fact was recorded. Source data age alone says nothing about
		// duplication, so it never drives the verdict.
		existing, err := matchExistingFingerprint(ctx, tx, doc.PlatformID, kind, in.ingestFingerprints())
		if err != nil {
			return nil, err
		}
		if existing != nil {
			interval := ws.Now().Sub(existing.CreatedAt)
			recordWin := time.Duration(settings.DuplicateRecordMinutes) * time.Minute
			askWin := time.Duration(settings.DuplicateAskDays) * 24 * time.Hour
			switch {
			case interval >= 0 && interval <= recordWin:
				return ws.recordObservation(ctx, tx, doc, in, existing.ID, domain.DuplicateRecordOnly, "within_record_window", true)
			case interval >= 0 && interval <= askWin:
				return ws.recordObservation(ctx, tx, doc, in, existing.ID, domain.DuplicateAskOperator, "within_ask_window", false)
			}
			// Beyond both windows the matching content is old enough to count
			// as a new responsibility; fall through to normal creation.
		}
		// First-seen content is a new responsibility regardless of age.
	}
	if _, err := ws.createFactWithLines(ctx, tx, doc, in, kind, nil); err != nil {
		return nil, err
	}
	return nil, nil
}

// recordObservation stores one duplicate observation. Undecided observations
// carry a JSON snapshot of the ingest input so a later operator decision can
// replay the fact creation when a new responsibility is claimed.
func (ws *Workspace) recordObservation(ctx context.Context, tx domain.Store, doc *domain.InputDocument, in IngestFactInput, existingFactID uint, verdict domain.DuplicateVerdict, reason string, decided bool) (*domain.DuplicateObservation, error) {
	obs := &domain.DuplicateObservation{DocumentID: doc.ID, ExistingFactID: existingFactID, Verdict: string(verdict), Reason: reason, Decided: decided}
	if !decided {
		snap, err := json.Marshal(duplicateInputSnapshot{Fact: in})
		if err != nil {
			return nil, err
		}
		obs.ExtraData = string(snap)
	}
	if err := tx.CreateDuplicate(ctx, obs); err != nil {
		return nil, err
	}
	return obs, nil
}

// duplicateInputSnapshot is the JSON payload stored on a duplicate
// observation's ExtraData so DecideDuplicate can rebuild the original input.
type duplicateInputSnapshot struct {
	Fact IngestFactInput `json:"fact"`
}

// createFactWithLines materializes one ingest input as a fact with its
// identity, lines, and alias-resolved product alignment. A non-nil revisesID
// marks the new fact as a revision of the established fact (which shares the
// stable external id, so it must be set at insert time). Every read and write
// goes through the explicit store argument so callers control the transaction.
func (ws *Workspace) createFactWithLines(ctx context.Context, tx domain.Store, doc *domain.InputDocument, in IngestFactInput, kind string, revisesID *uint) (*domain.InputFact, error) {
	var identID *uint
	if in.IdentityValue != "" {
		typ := in.IdentityType
		if typ == "" {
			typ = string(domain.IdentityTypePlatformUID)
		}
		norm := NormalizeIdentity(in.IdentityValue)
		ident, err := tx.FindIdentity(ctx, doc.PlatformID, typ, norm)
		if err == domain.ErrNotFound {
			ident = &domain.PlatformIdentity{PlatformID: doc.PlatformID, IdentityType: typ, IdentityValue: in.IdentityValue, NormalizedValue: norm}
			if err := tx.CreateIdentity(ctx, ident); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		identID = &ident.ID
	}

	docID := doc.ID
	fact := &domain.InputFact{
		DocumentID:         &docID,
		PlatformID:         doc.PlatformID,
		Kind:               kind,
		StableExternalID:   in.StableExternalID,
		CustomerProfileID:  in.CustomerProfileID,
		PlatformIdentityID: identID,
		MembershipLevel:    in.MembershipLevel,
		SourceDocumentNo:   in.SourceDocumentNo,
		SourceCreatedAt:    in.SourceCreatedAt,
		RevisesID:          revisesID,
		ExtraData:          in.ExtraData,
	}
	if identID != nil {
		ident, err := tx.GetIdentity(ctx, *identID)
		if err != nil {
			return nil, err
		}
		fact.CustomerProfileID = ident.CustomerProfileID
	}
	if err := tx.CreateFact(ctx, fact); err != nil {
		return nil, err
	}
	lines := in.Lines
	if kind == string(domain.InputFactKindMembership) || kind == string(domain.InputFactKindOperatorGrant) {
		if len(lines) == 0 {
			lines = []IngestLine{{SourceLineNo: 1, Quantity: 1}}
		}
		if len(lines) != 1 {
			return nil, fmt.Errorf("%s fact must have exactly one line", kind)
		}
	}
	for _, ln := range lines {
		qty := ln.Quantity
		if qty == 0 {
			qty = 1
		}
		line := &domain.InputFactLine{
			FactID:        fact.ID,
			SourceLineNo:  ln.SourceLineNo,
			ExternalSKU:   ln.ExternalSKU,
			ExternalTitle: ln.ExternalTitle,
			ExternalSpec:  ln.ExternalSpec,
			Quantity:      qty,
		}
		if ln.ExternalSKU != "" {
			if alias, err := tx.FindAlias(ctx, doc.PlatformID, ln.ExternalSKU); err == nil {
				id := alias.ProductItemID
				line.ProductItemID = &id
			} else if err != domain.ErrNotFound {
				return nil, err
			}
		}
		if err := tx.CreateFactLine(ctx, line); err != nil {
			return nil, err
		}
	}
	return fact, nil
}

func (ws *Workspace) AttachIdentity(ctx context.Context, identityID, customerID uint) error {
	ident, err := ws.Store.GetIdentity(ctx, identityID)
	if err != nil {
		return err
	}
	ident.CustomerProfileID = &customerID
	if err := ws.Store.UpdateIdentity(ctx, ident); err != nil {
		return err
	}
	return nil
}

func (ws *Workspace) AssignLines(ctx context.Context, waveID uint, lineIDs []uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		tws := ws.withStore(tx)
		if err := tws.assignLines(ctx, waveID, lineIDs); err != nil {
			return err
		}
		return recompute(ctx, tx, waveID)
	})
}

// assignLines assigns fact lines into a wave and creates the matching
// instances and source results. It must run on a workspace bound to the
// surrounding transaction; the caller recompute runs in the same transaction.
func (ws *Workspace) assignLines(ctx context.Context, waveID uint, lineIDs []uint) error {
	wave, err := ws.Store.GetWave(ctx, waveID)
	if err != nil {
		return err
	}
	if wave.CloseResult != string(domain.WaveCloseResultOpen) {
		return ErrWaveClosed
	}
	for _, id := range lineIDs {
		line, err := ws.Store.GetFactLine(ctx, id)
		if err != nil {
			return err
		}
		if line.WaveID != nil && *line.WaveID != waveID {
			return ErrAlreadyAssigned
		}
		fact, err := ws.Store.GetFact(ctx, line.FactID)
		if err != nil {
			return err
		}
		if fact.RevisesID != nil && fact.RevisionAppliedAt == nil {
			return ErrRevisionPending
		}
		wid := waveID
		line.WaveID = &wid
		if err := ws.Store.UpdateFactLine(ctx, line); err != nil {
			return err
		}
		if fact.Kind == string(domain.InputFactKindMembership) {
			if _, err := ws.Store.GetInstanceByLine(ctx, waveID, line.ID); err == domain.ErrNotFound {
				inst := &domain.EntitlementInstance{
					WaveID:             waveID,
					InputFactLineID:    line.ID,
					CustomerProfileID:  fact.CustomerProfileID,
					PlatformIdentityID: fact.PlatformIdentityID,
					MembershipLevel:    fact.MembershipLevel,
				}
				if err := ws.Store.CreateInstance(ctx, inst); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
		if fact.Kind == string(domain.InputFactKindRetailOrder) {
			if err := ws.ensureRetailResult(ctx, waveID, fact, line); err != nil {
				return err
			}
		}
		if fact.Kind == string(domain.InputFactKindOperatorGrant) {
			if err := ws.ensureGrantResult(ctx, waveID, fact, line); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ws *Workspace) ensureRetailResult(ctx context.Context, waveID uint, fact *domain.InputFact, line *domain.InputFactLine) error {
	results, err := ws.Store.ListResults(ctx, waveID)
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.InputFactLineID != nil && *r.InputFactLineID == line.ID && r.SourceKind == string(domain.SourceRetailLine) {
			return nil
		}
	}
	custID := fact.CustomerProfileID
	if custID == nil && fact.PlatformIdentityID != nil {
		if ident, err := ws.Store.GetIdentity(ctx, *fact.PlatformIdentityID); err == nil && ident.CustomerProfileID != nil {
			custID = ident.CustomerProfileID
		}
	}
	addr, err := defaultSnapshot(ctx, ws.Store, custID)
	if err != nil {
		return err
	}
	fid := fact.ID
	lid := line.ID

	// Bundle mapping: when the line's alias expands into components, emit one
	// result per component with quantity = line quantity x component quantity.
	if comps, ok, err := ws.bundleComponentsForLine(ctx, fact, line); err != nil {
		return err
	} else if ok {
		for _, comp := range comps {
			pid := comp.ProductItemID
			res := &domain.FulfillmentResult{
				WaveID:            waveID,
				SourceKind:        string(domain.SourceRetailLine),
				InputFactLineID:   &lid,
				InputFactID:       &fid,
				CustomerProfileID: custID,
				ProductItemID:     &pid,
				Quantity:          line.Quantity * comp.Quantity,
				Address:           addr,
				ExtraData:         fmt.Sprintf(`{"bundle_alias_line":%d}`, line.ID),
			}
			if err := ws.Store.CreateResult(ctx, res); err != nil {
				return err
			}
		}
		return nil
	}

	res := &domain.FulfillmentResult{
		WaveID:            waveID,
		SourceKind:        string(domain.SourceRetailLine),
		InputFactLineID:   &lid,
		InputFactID:       &fid,
		CustomerProfileID: custID,
		ProductItemID:     line.ProductItemID,
		Quantity:          line.Quantity,
		Address:           addr,
	}
	return ws.Store.CreateResult(ctx, res)
}

// bundleComponentsForLine resolves the line's external SKU to its alias and
// returns the alias's bundle components, if any. ok=false means the alias is
// not a bundle (plain alignment applies).
func (ws *Workspace) bundleComponentsForLine(ctx context.Context, fact *domain.InputFact, line *domain.InputFactLine) ([]domain.ProductBundleComponent, bool, error) {
	if line.ExternalSKU == "" {
		return nil, false, nil
	}
	alias, err := ws.Store.FindAlias(ctx, fact.PlatformID, line.ExternalSKU)
	if err == domain.ErrNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	comps, err := ws.Store.ListBundleComponents(ctx, alias.ID)
	if err != nil {
		return nil, false, err
	}
	if len(comps) == 0 {
		return nil, false, nil
	}
	return comps, true, nil
}

func (ws *Workspace) ensureGrantResult(ctx context.Context, waveID uint, fact *domain.InputFact, line *domain.InputFactLine) error {
	results, err := ws.Store.ListResults(ctx, waveID)
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.InputFactLineID != nil && *r.InputFactLineID == line.ID && r.SourceKind == string(domain.SourceOperatorGrant) {
			return nil
		}
	}
	addr, err := defaultSnapshot(ctx, ws.Store, fact.CustomerProfileID)
	if err != nil {
		return err
	}
	fid := fact.ID
	lid := line.ID
	res := &domain.FulfillmentResult{
		WaveID:            waveID,
		SourceKind:        string(domain.SourceOperatorGrant),
		InputFactLineID:   &lid,
		InputFactID:       &fid,
		CustomerProfileID: fact.CustomerProfileID,
		ProductItemID:     line.ProductItemID,
		Quantity:          line.Quantity,
		Address:           addr,
	}
	return ws.Store.CreateResult(ctx, res)
}

func (ws *Workspace) ListInboxRows(ctx context.Context) ([]InboxRow, error) {
	docs, err := ws.Store.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}
	docByID := map[uint]domain.InputDocument{}
	for _, d := range docs {
		docByID[d.ID] = d
	}
	var rows []InboxRow
	for _, d := range docs {
		facts, err := ws.Store.ListFactsByDocument(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		for _, f := range facts {
			lines, err := ws.Store.ListFactLines(ctx, f.ID)
			if err != nil {
				return nil, err
			}
			docCopy := d
			unattached := false
			if f.PlatformIdentityID != nil {
				ident, err := ws.Store.GetIdentity(ctx, *f.PlatformIdentityID)
				if err != nil {
					return nil, err
				}
				unattached = ident.CustomerProfileID == nil
			}
			revisionPending := f.RevisesID != nil && f.RevisionAppliedAt == nil
			for _, ln := range lines {
				var aliasID *uint
				if ln.ExternalSKU != "" {
					if alias, err := ws.Store.FindAlias(ctx, f.PlatformID, ln.ExternalSKU); err == nil {
						id := alias.ID
						aliasID = &id
					} else if err != domain.ErrNotFound {
						return nil, err
					}
				}
				row := InboxRow{
					Line:            ln,
					Fact:            f,
					Document:        &docCopy,
					Assigned:        ln.WaveID != nil,
					Unaligned:       ln.ProductItemID == nil && f.Kind != string(domain.InputFactKindMembership),
					Unattached:      unattached,
					RevisionPending: revisionPending,
					AliasID:         aliasID,
				}
				rows = append(rows, row)
			}
		}
	}
	return rows, nil
}

func (ws *Workspace) DecideDuplicate(ctx context.Context, id uint, accept bool) error {
	open, err := ws.Store.ListOpenDuplicates(ctx)
	if err != nil {
		return err
	}
	for i := range open {
		if open[i].ID == id {
			open[i].Decided = true
			if accept {
				open[i].Verdict = string(domain.DuplicateRecordOnly)
			} else {
				open[i].Verdict = string(domain.DuplicateNewResponsibility)
			}
			return ws.Store.UpdateDuplicate(ctx, &open[i])
		}
	}
	return domain.ErrNotFound
}
