package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// setupDuplicateWindowWorkspace builds a workspace with its backing db handle
// (used to rewind created_at) and the source platform.
func setupDuplicateWindowWorkspace(t *testing.T) (*Workspace, *gorm.DB, *domain.Platform) {
	t.Helper()
	gdb := openTestDB(t)
	ws := NewWorkspace(infra.NewGormStore(gdb))
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatalf("EnsureBuiltinPlatforms: %v", err)
	}
	for _, p := range mustListPlatforms(t, ws) {
		if p.Kind == string(domain.PlatformKindSource) {
			return ws, gdb, &p
		}
	}
	t.Fatal("no source platform")
	return nil, nil, nil
}

func mustListPlatforms(t *testing.T, ws *Workspace) []domain.Platform {
	t.Helper()
	platforms, err := ws.ListPlatforms(context.Background())
	if err != nil {
		t.Fatalf("ListPlatforms: %v", err)
	}
	return platforms
}

func mustFactsByPlatform(t *testing.T, ws *Workspace, platformID uint) []domain.InputFact {
	t.Helper()
	facts, err := ws.Store.ListFactsByPlatform(context.Background(), platformID)
	if err != nil {
		t.Fatalf("ListFactsByPlatform: %v", err)
	}
	return facts
}

func rewindFactCreatedAt(t *testing.T, gdb *gorm.DB, factID uint, to time.Time) {
	t.Helper()
	if err := gdb.Exec("UPDATE input_facts SET created_at = ? WHERE id = ?", to, factID).Error; err != nil {
		t.Fatalf("rewind input_facts.created_at: %v", err)
	}
}

func TestDuplicateWindowFourQuadrants(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.Local)
	srcAt := now.Add(48 * time.Hour)

	run := func(name string, rewind time.Duration) {
		t.Run(name, func(t *testing.T) {
			ws, gdb, source := setupDuplicateWindowWorkspace(t)
			ws.Now = func() time.Time { return now }
			ctx := context.Background()

			base := IngestFactInput{
				Kind:            string(domain.InputFactKindRetailOrder),
				IdentityType:    string(domain.IdentityTypePlatformUID),
				IdentityValue:   "UID-W1",
				SourceCreatedAt: &srcAt,
				Lines:           []IngestLine{{SourceLineNo: 1, ExternalSKU: "SKU-W", Quantity: 2}},
			}
			if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail", OriginalName: "a.csv"}, []IngestFactInput{base}); err != nil {
				t.Fatalf("first ingest: %v", err)
			} else if len(dups) != 0 {
				t.Fatalf("first ingest produced duplicates: %v", dups)
			}
			facts := mustFactsByPlatform(t, ws, source.ID)
			if len(facts) != 1 {
				t.Fatalf("expected 1 fact after first ingest, got %d", len(facts))
			}
			rewindFactCreatedAt(t, gdb, facts[0].ID, now.Add(-rewind))

			_, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail", OriginalName: "b.csv"}, []IngestFactInput{base})
			if err != nil {
				t.Fatalf("second ingest: %v", err)
			}
			after := mustFactsByPlatform(t, ws, source.ID)
			switch name {
			case "hit short interval records only":
				if len(dups) != 1 || dups[0].Verdict != string(domain.DuplicateRecordOnly) || !dups[0].Decided {
					t.Fatalf("expected one decided record_only observation, got %+v", dups)
				}
				if dups[0].ExistingFactID != facts[0].ID {
					t.Fatalf("observation must reference existing fact %d, got %d", facts[0].ID, dups[0].ExistingFactID)
				}
				if dups[0].Reason != "within_record_window" {
					t.Fatalf("reason = %q, want within_record_window", dups[0].Reason)
				}
				if len(after) != 1 {
					t.Fatalf("record_only must not create a fact, got %d facts", len(after))
				}
			case "hit middle interval asks operator":
				if len(dups) != 1 || dups[0].Verdict != string(domain.DuplicateAskOperator) || dups[0].Decided {
					t.Fatalf("expected one undecided ask_operator observation, got %+v", dups)
				}
				if dups[0].ExistingFactID != facts[0].ID {
					t.Fatalf("observation must reference existing fact %d, got %d", facts[0].ID, dups[0].ExistingFactID)
				}
				var snap duplicateInputSnapshot
				if err := json.Unmarshal([]byte(dups[0].ExtraData), &snap); err != nil {
					t.Fatalf("ask observation must carry an input snapshot, got ExtraData %q: %v", dups[0].ExtraData, err)
				}
				if snap.Fact.IdentityValue != "UID-W1" || len(snap.Fact.Lines) != 1 {
					t.Fatalf("snapshot content = %+v", snap.Fact)
				}
				if len(after) != 1 {
					t.Fatalf("ask_operator must not create a fact, got %d facts", len(after))
				}
			case "hit beyond window creates responsibility":
				if len(dups) != 0 {
					t.Fatalf("expected no observation beyond both windows, got %+v", dups)
				}
				if len(after) != 2 {
					t.Fatalf("expected a second fact, got %d", len(after))
				}
			}
		})
	}

	run("hit short interval records only", 5*time.Minute)
	run("hit middle interval asks operator", 48*time.Hour)
	run("hit beyond window creates responsibility", 20*24*time.Hour)

	t.Run("no hit creates responsibility regardless of source age", func(t *testing.T) {
		ws, _, source := setupDuplicateWindowWorkspace(t)
		ws.Now = func() time.Time { return now }
		ctx := context.Background()
		ancient := now.Add(-365 * 24 * time.Hour)

		first := IngestFactInput{
			Kind: string(domain.InputFactKindRetailOrder), IdentityType: string(domain.IdentityTypePlatformUID),
			IdentityValue: "UID-W2", SourceCreatedAt: &ancient,
			Lines: []IngestLine{{SourceLineNo: 1, ExternalSKU: "SKU-A", Quantity: 1}},
		}
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{first}); err != nil {
			t.Fatalf("first ingest: %v", err)
		} else if len(dups) != 0 {
			t.Fatalf("first ingest produced duplicates: %v", dups)
		}

		// Same identity and day but different quantity: no fingerprint hit,
		// and an ancient source creation time must not matter.
		second := first
		second.Lines = []IngestLine{{SourceLineNo: 1, ExternalSKU: "SKU-A", Quantity: 9}}
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{second}); err != nil {
			t.Fatalf("second ingest: %v", err)
		} else if len(dups) != 0 {
			t.Fatalf("first-seen content must be a new responsibility, got %+v", dups)
		}
		if facts := mustFactsByPlatform(t, ws, source.ID); len(facts) != 2 {
			t.Fatalf("expected 2 facts, got %d", len(facts))
		}

		// Identical content but a different source day: different fingerprint
		// (the day is part of the construction), so still a new responsibility.
		third := first
		day := now.Add(-180 * 24 * time.Hour)
		third.SourceCreatedAt = &day
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{third}); err != nil {
			t.Fatalf("third ingest: %v", err)
		} else if len(dups) != 0 {
			t.Fatalf("different-day content must not match, got %+v", dups)
		}
		if facts := mustFactsByPlatform(t, ws, source.ID); len(facts) != 3 {
			t.Fatalf("expected 3 facts, got %d", len(facts))
		}
	})

	t.Run("same content different order number is a new responsibility", func(t *testing.T) {
		ws, gdb, source := setupDuplicateWindowWorkspace(t)
		ws.Now = func() time.Time { return now }
		ctx := context.Background()
		day := now.Add(-24 * time.Hour)

		base := IngestFactInput{
			Kind:             string(domain.InputFactKindRetailOrder),
			IdentityType:     string(domain.IdentityTypePlatformUID),
			IdentityValue:    "UID-ORDNO",
			SourceDocumentNo: "ORD-1001",
			SourceCreatedAt:  &day,
			Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "SKU-ORDNO", Quantity: 1}},
		}
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{base}); err != nil {
			t.Fatalf("first ingest: %v", err)
		} else if len(dups) != 0 {
			t.Fatalf("first ingest produced duplicates: %v", dups)
		}
		facts := mustFactsByPlatform(t, ws, source.ID)
		// Park the first fact inside the record window so a genuine content
		// repeat would be swallowed as record_only.
		rewindFactCreatedAt(t, gdb, facts[0].ID, now.Add(-time.Minute))

		// Same person, product, quantity, and day, but a different order
		// number: a second genuine order, not a swallowed duplicate.
		second := base
		second.SourceDocumentNo = "ORD-1002"
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{second}); err != nil {
			t.Fatalf("second ingest: %v", err)
		} else if len(dups) != 0 {
			t.Fatalf("a different order number must not be judged duplicate, got %+v", dups)
		}
		after := mustFactsByPlatform(t, ws, source.ID)
		if len(after) != 2 {
			t.Fatalf("expected 2 facts (one per real order), got %d", len(after))
		}

		// The same order number repeated still hits the window logic.
		for _, fact := range after {
			rewindFactCreatedAt(t, gdb, fact.ID, now.Add(-time.Minute))
		}
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{second}); err != nil {
			t.Fatalf("repeat ingest: %v", err)
		} else if len(dups) != 1 || dups[0].Verdict != string(domain.DuplicateRecordOnly) {
			t.Fatalf("repeated order number must stay record_only, got %+v", dups)
		}
	})

	t.Run("membership fingerprints identity and day", func(t *testing.T) {
		ws, gdb, source := setupDuplicateWindowWorkspace(t)
		ws.Now = func() time.Time { return now }
		ctx := context.Background()
		day := now.Add(-24 * time.Hour)

		mem := IngestFactInput{
			Kind: string(domain.InputFactKindMembership), IdentityType: string(domain.IdentityTypePlatformUID),
			IdentityValue: "UID-W3", MembershipLevel: "captain", SourceCreatedAt: &day,
		}
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{mem}); err != nil {
			t.Fatalf("first ingest: %v", err)
		} else if len(dups) != 0 {
			t.Fatalf("first ingest produced duplicates: %v", dups)
		}
		facts := mustFactsByPlatform(t, ws, source.ID)
		rewindFactCreatedAt(t, gdb, facts[0].ID, now.Add(-2*time.Minute))

		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{mem}); err != nil {
			t.Fatalf("second ingest: %v", err)
		} else if len(dups) != 1 || dups[0].Verdict != string(domain.DuplicateRecordOnly) {
			t.Fatalf("expected record_only on same-day membership re-import, got %+v", dups)
		}

		// Different identity value: no hit, new responsibility.
		other := mem
		other.IdentityValue = "UID-W4"
		if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID}, []IngestFactInput{other}); err != nil {
			t.Fatalf("third ingest: %v", err)
		} else if len(dups) != 0 {
			t.Fatalf("different identity must not match, got %+v", dups)
		}
		if facts := mustFactsByPlatform(t, ws, source.ID); len(facts) != 2 {
			t.Fatalf("expected 2 facts, got %d", len(facts))
		}
	})
}
