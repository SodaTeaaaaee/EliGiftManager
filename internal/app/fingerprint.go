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
// identity type, normalized identity value, product alias, quantity, and the
// source creation day. The hashing itself reuses alignment.FingerprintValues
// so template-driven fingerprints and ingest-time fingerprints share one
// mechanism.
var fingerprintKeys = []string{
	"identity.type",
	"identity.value",
	"product.alias_id",
	"quantity",
	"source.created_at",
}

// factFingerprint hashes one content combination into the duplicate-detection
// fingerprint.
func factFingerprint(identityType, normalizedIdentity, aliasSKU string, qty int, sourceCreatedAt *time.Time) string {
	day := ""
	if sourceCreatedAt != nil {
		day = sourceCreatedAt.Format("2006-01-02")
	}
	return alignment.FingerprintValues(map[string]string{
		"identity.type":     identityType,
		"identity.value":    normalizedIdentity,
		"product.alias_id":  aliasSKU,
		"quantity":          strconv.Itoa(qty),
		"source.created_at": day,
	}, fingerprintKeys)
}

// ingestFingerprints returns one fingerprint per effective fact line, applying
// the same single-line default and quantity floor the line creation applies.
func (in IngestFactInput) ingestFingerprints() []string {
	typ := in.IdentityType
	if typ == "" {
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
		fps = append(fps, factFingerprint(typ, norm, ln.ExternalSKU, qty, in.SourceCreatedAt))
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
		fps = append(fps, factFingerprint(typ, norm, ln.ExternalSKU, ln.Quantity, fact.SourceCreatedAt))
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
