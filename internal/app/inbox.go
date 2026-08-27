package app

import (
	"context"
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
	Lines             []IngestLine
}

type InboxRow struct {
	Line       domain.InputFactLine
	Fact       domain.InputFact
	Document   *domain.InputDocument
	Assigned   bool
	Unaligned  bool
	Unattached bool
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
	if in.StableExternalID != "" {
		existing, err := tx.FindFactByStableID(ctx, doc.PlatformID, in.StableExternalID)
		if err == nil {
			obs := &domain.DuplicateObservation{DocumentID: doc.ID, ExistingFactID: existing.ID, Verdict: string(domain.DuplicateRecordOnly), Reason: "stable_external_id", Decided: true}
			if err := tx.CreateDuplicate(ctx, obs); err != nil {
				return nil, err
			}
			return obs, nil
		}
		if err != domain.ErrNotFound {
			return nil, err
		}
	} else if in.SourceCreatedAt != nil {
		facts, err := tx.ListFactsByDocument(ctx, doc.ID)
		if err != nil {
			return nil, err
		}
		_ = facts
		age := ws.Now().Sub(*in.SourceCreatedAt)
		recordWin := time.Duration(settings.DuplicateRecordMinutes) * time.Minute
		askWin := time.Duration(settings.DuplicateAskDays) * 24 * time.Hour
		if age >= 0 && age <= recordWin {
			obs := &domain.DuplicateObservation{DocumentID: doc.ID, Verdict: string(domain.DuplicateRecordOnly), Reason: "within_record_window", Decided: true}
			if err := tx.CreateDuplicate(ctx, obs); err != nil {
				return nil, err
			}
			return obs, nil
		}
		if age > recordWin && age <= askWin {
			obs := &domain.DuplicateObservation{DocumentID: doc.ID, Verdict: string(domain.DuplicateAskOperator), Reason: "within_ask_window", Decided: false}
			if err := tx.CreateDuplicate(ctx, obs); err != nil {
				return nil, err
			}
			return obs, nil
		}
	}

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

	kind := in.Kind
	if kind == "" {
		kind = string(domain.InputFactKindRetailOrder)
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
	return nil, nil
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
		return tws.recompute(ctx, tx, waveID)
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
	addr, err := ws.defaultSnapshot(ctx, custID)
	if err != nil {
		return err
	}
	fid := fact.ID
	lid := line.ID
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
	addr, err := ws.defaultSnapshot(ctx, fact.CustomerProfileID)
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
			for _, ln := range lines {
				row := InboxRow{Line: ln, Fact: f, Document: &docCopy, Assigned: ln.WaveID != nil, Unaligned: ln.ProductItemID == nil && f.Kind != string(domain.InputFactKindMembership), Unattached: unattached}
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
