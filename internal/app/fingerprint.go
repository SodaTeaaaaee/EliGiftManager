package app

import (
	"context"
	"strconv"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// fingerprintKeys is the fixed content-fingerprint construction used for
// suspected-duplicate detection when no stable external fact id exists:
// identity type, normalized identity value, product alias, quantity, the
// source order number, and the source creation day. The order number is part
// of the content: two genuine orders of the same person, product, and day
// must not collapse into one (membership and grant facts carry no order
// number, so the empty slot adds no discrimination there). The hashing itself
// reuses alignment.FingerprintValues so template-driven fingerprints and
// ingest-time fingerprints share one mechanism.
var fingerprintKeys = []string{
	"identity.type",
	"identity.value",
	"product.alias_id",
	"quantity",
	"source.document_no",
	"source.created_at",
}

// factFingerprint hashes one content combination into the duplicate-detection
// fingerprint.
func factFingerprint(identityType, normalizedIdentity, aliasSKU, sourceDocNo string, qty int, sourceCreatedAt *time.Time) string {
	day := ""
	if sourceCreatedAt != nil {
		day = sourceCreatedAt.Format("2006-01-02")
	}
	return alignment.FingerprintValues(map[string]string{
		"identity.type":      identityType,
		"identity.value":     normalizedIdentity,
		"product.alias_id":   aliasSKU,
		"quantity":           strconv.Itoa(qty),
		"source.document_no": sourceDocNo,
		"source.created_at":  day,
	}, fingerprintKeys)
}

// ingestFingerprints returns one fingerprint per effective fact line, applying
// the same single-line default and quantity floor the line creation applies.
// An empty identity value hashes without a type: the type default is a
// creation-time convenience, not content, and stored identity-less facts
// (which never carry an identity row) hash the same way.
func (in IngestFactInput) ingestFingerprints() []string {
	typ := in.IdentityType
	if in.IdentityValue == "" {
		typ = ""
	} else if typ == "" {
		typ = string(domain.IdentityTypePlatformUID)
	}
	norm := NormalizeIdentity(in.IdentityValue)
	lines := in.Lines
	if len(lines) == 0 {
		lines = []IngestLine{{Quantity: 1}}
	}
	fps := make([]string, 0, len(lines))
	for _, ln := range lines {
		qty := ln.Quantity
		if qty == 0 {
			qty = 1
		}
		fps = append(fps, factFingerprint(typ, norm, ln.ExternalSKU, in.SourceDocumentNo, qty, in.SourceCreatedAt))
	}
	return fps
}

// factLineFingerprints computes the same fingerprints for a stored fact: one
// per line, with the identity pulled from the stored platform identity.
func factLineFingerprints(ctx context.Context, store domain.Store, fact domain.InputFact) ([]string, error) {
	typ, norm := "", ""
	if fact.PlatformIdentityID != nil {
		ident, err := store.GetIdentity(ctx, *fact.PlatformIdentityID)
		if err != nil {
			if err == domain.ErrNotFound {
				ident = nil
			} else {
				return nil, err
			}
		}
		if ident != nil {
			typ, norm = ident.IdentityType, ident.NormalizedValue
		}
	}
	lines, err := store.ListFactLines(ctx, fact.ID)
	if err != nil {
		return nil, err
	}
	fps := make([]string, 0, len(lines))
	for _, ln := range lines {
		fps = append(fps, factFingerprint(typ, norm, ln.ExternalSKU, fact.SourceDocumentNo, ln.Quantity, fact.SourceCreatedAt))
	}
	return fps, nil
}

// matchExistingFingerprint finds the first stored fact of the platform and
// kind whose lines share any fingerprint with fps. Pending revision facts are
// not duplicate anchors: they are versions of an established fact, not
// independent content.
func matchExistingFingerprint(ctx context.Context, store domain.Store, platformID uint, kind string, fps []string) (*domain.InputFact, error) {
	fpSet := make(map[string]struct{}, len(fps))
	for _, fp := range fps {
		fpSet[fp] = struct{}{}
	}
	facts, err := store.ListFactsByPlatform(ctx, platformID)
	if err != nil {
		return nil, err
	}
	for i := range facts {
		fact := &facts[i]
		if fact.Kind != kind || fact.RevisesID != nil {
			continue
		}
		existing, err := factLineFingerprints(ctx, store, *fact)
		if err != nil {
			return nil, err
		}
		for _, fp := range existing {
			if _, hit := fpSet[fp]; hit {
				return fact, nil
			}
		}
	}
	return nil, nil
}

// sameContentFingerprints reports whether a stored fact's line fingerprints
// form the same multiset as fps — the boundary between "same content again"
// (duplicate) and "corrected content" (revision). Repeated fingerprints are
// counted, not just contained: [X, X] is not the same content as [X, Y].
func sameContentFingerprints(ctx context.Context, store domain.Store, fact domain.InputFact, fps []string) (bool, error) {
	existing, err := factLineFingerprints(ctx, store, fact)
	if err != nil {
		return false, err
	}
	if len(existing) != len(fps) {
		return false, nil
	}
	counts := make(map[string]int, len(existing))
	for _, fp := range existing {
		counts[fp]++
	}
	for _, fp := range fps {
		counts[fp]--
		if counts[fp] < 0 {
			return false, nil
		}
	}
	return true, nil
}
