package controller

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDemandInboxTestDB spins up an in-memory sqlite DB migrated with only the tables the
// demand inbox query touches.
func setupDemandInboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := gdb.AutoMigrate(
		&persistence.DemandDocument{},
		&persistence.DemandLine{},
		&persistence.Wave{},
		&persistence.WaveDemandAssignment{},
		&persistence.IntegrationProfile{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return gdb
}

// newDemandInboxTestController builds a DemandController wired to the given in-memory DB,
// bypassing NewDemandController's db.GetDB() singleton lookup.
func newDemandInboxTestController(gdb *gorm.DB) *DemandController {
	demandRepo := infra.NewDemandRepository(gdb)
	integrationProfileRepo := infra.NewIntegrationProfileRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	return &DemandController{
		demandRepo:          demandRepo,
		integrationProfile:  integrationProfileRepo,
		assignmentRepo:      assignmentRepo,
		waveRepo:            waveRepo,
		inboxAssignmentRepo: infra.NewDemandInboxAssignmentRepository(gdb),
		inboxLineRepo:       infra.NewDemandInboxLineRepository(gdb),
	}
}

// seedDemandInboxFixture creates two integration profiles, three demand documents (two bound
// to profile A, one to profile B), a wave, an assignment linking one document to that wave,
// and a handful of demand lines with varied routing dispositions to exercise the row rollup.
func seedDemandInboxFixture(t *testing.T, gdb *gorm.DB) (profileAID, profileBID uint, waveID uint, docIDs []uint) {
	t.Helper()
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profileA := &domain.IntegrationProfile{ProfileKey: "patreon-membership", SourceChannel: "patreon"}
	if err := profileRepo.Create(ctx, profileA); err != nil {
		t.Fatalf("create profile A: %v", err)
	}
	profileB := &domain.IntegrationProfile{ProfileKey: "bilibili-shop", SourceChannel: "bilibili"}
	if err := profileRepo.Create(ctx, profileB); err != nil {
		t.Fatalf("create profile B: %v", err)
	}

	waveRepo := infra.NewWaveRepository(gdb)
	wave := &domain.Wave{WaveNo: "W-2026-07", Name: "July Wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}

	demandRepo := infra.NewDemandRepository(gdb)

	doc1 := &domain.DemandDocument{Kind: "membership_entitlement", CaptureMode: "document_import", IntegrationProfileID: &profileA.ID}
	if err := demandRepo.Create(ctx, doc1); err != nil {
		t.Fatalf("create doc1: %v", err)
	}
	doc2 := &domain.DemandDocument{Kind: "membership_entitlement", CaptureMode: "document_import", IntegrationProfileID: &profileA.ID}
	if err := demandRepo.Create(ctx, doc2); err != nil {
		t.Fatalf("create doc2: %v", err)
	}
	doc3 := &domain.DemandDocument{Kind: "shop_order", CaptureMode: "document_import", IntegrationProfileID: &profileB.ID}
	if err := demandRepo.Create(ctx, doc3); err != nil {
		t.Fatalf("create doc3: %v", err)
	}

	// doc1 gets two lines: one accepted+ready, one deferred.
	if err := demandRepo.CreateLine(ctx, &domain.DemandLine{
		DemandDocumentID: doc1.ID, LineType: "gift", RoutingDisposition: "accepted", RecipientInputState: "ready",
	}); err != nil {
		t.Fatalf("create doc1 line1: %v", err)
	}
	if err := demandRepo.CreateLine(ctx, &domain.DemandLine{
		DemandDocumentID: doc1.ID, LineType: "gift", RoutingDisposition: "deferred",
	}); err != nil {
		t.Fatalf("create doc1 line2: %v", err)
	}
	// doc2 gets one excluded line.
	if err := demandRepo.CreateLine(ctx, &domain.DemandLine{
		DemandDocumentID: doc2.ID, LineType: "gift", RoutingDisposition: "excluded_manual",
	}); err != nil {
		t.Fatalf("create doc2 line1: %v", err)
	}
	// doc3 gets one accepted+waiting line.
	if err := demandRepo.CreateLine(ctx, &domain.DemandLine{
		DemandDocumentID: doc3.ID, LineType: "gift", RoutingDisposition: "accepted", RecipientInputState: "waiting_for_input",
	}); err != nil {
		t.Fatalf("create doc3 line1: %v", err)
	}

	// Assign doc1 to the wave; doc2 and doc3 remain unassigned.
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	if err := assignmentRepo.Create(ctx, &domain.WaveDemandAssignment{WaveID: wave.ID, DemandDocumentID: doc1.ID}); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	return profileA.ID, profileB.ID, wave.ID, []uint{doc1.ID, doc2.ID, doc3.ID}
}

func TestListDemandInboxRows_PaginationAndAssembly(t *testing.T) {
	gdb := setupDemandInboxTestDB(t)
	_, _, waveID, docIDs := seedDemandInboxFixture(t, gdb)
	c := newDemandInboxTestController(gdb)

	// Page size 2, page 1: expect 2 of the 3 documents, correct total count/pages.
	result, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{}, dto.PaginationInput{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListDemandInboxRows page 1: %v", err)
	}
	if result.Pagination.TotalCount != 3 {
		t.Fatalf("expected totalCount 3, got %d", result.Pagination.TotalCount)
	}
	if result.Pagination.TotalPages != 2 {
		t.Fatalf("expected totalPages 2, got %d", result.Pagination.TotalPages)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows on page 1, got %d", len(result.Rows))
	}

	// Page 2 should hold the remaining 1 document — no duplicates, no missing rows.
	result2, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{}, dto.PaginationInput{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListDemandInboxRows page 2: %v", err)
	}
	if len(result2.Rows) != 1 {
		t.Fatalf("expected 1 row on page 2, got %d", len(result2.Rows))
	}

	seen := map[uint]bool{}
	for _, row := range append(append([]dto.DemandInboxRowDTO{}, result.Rows...), result2.Rows...) {
		if seen[row.DemandDocumentID] {
			t.Fatalf("document %d appeared more than once across pages", row.DemandDocumentID)
		}
		seen[row.DemandDocumentID] = true
	}
	for _, id := range docIDs {
		if !seen[id] {
			t.Fatalf("document %d missing from paginated results", id)
		}
	}

	// Row assembly correctness: locate doc1's row (fetch full unpaginated set to avoid
	// depending on ordering) and verify assignment + line rollups.
	full, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{}, dto.PaginationInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListDemandInboxRows full: %v", err)
	}
	var doc1Row *dto.DemandInboxRowDTO
	for i := range full.Rows {
		if full.Rows[i].DemandDocumentID == docIDs[0] {
			doc1Row = &full.Rows[i]
		}
	}
	if doc1Row == nil {
		t.Fatalf("doc1 row not found in full result")
	}
	if !doc1Row.Assigned || doc1Row.AssignedWaveID == nil || *doc1Row.AssignedWaveID != waveID {
		t.Fatalf("doc1 should be assigned to wave %d, got %+v", waveID, doc1Row)
	}
	if doc1Row.TotalLineCount != 2 || doc1Row.AcceptedCount != 1 || doc1Row.ReadyAcceptedCount != 1 || doc1Row.DeferredCount != 1 {
		t.Fatalf("doc1 line rollup mismatch: %+v", doc1Row)
	}
}

func TestListDemandInboxRows_ServerSideProfileFilter(t *testing.T) {
	gdb := setupDemandInboxTestDB(t)
	profileAID, profileBID, _, _ := seedDemandInboxFixture(t, gdb)
	c := newDemandInboxTestController(gdb)

	resultA, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{IntegrationProfileID: &profileAID}, dto.PaginationInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListDemandInboxRows profile A: %v", err)
	}
	if resultA.Pagination.TotalCount != 2 {
		t.Fatalf("expected 2 documents for profile A, got %d", resultA.Pagination.TotalCount)
	}
	for _, row := range resultA.Rows {
		if row.IntegrationProfileID == nil || *row.IntegrationProfileID != profileAID {
			t.Fatalf("row leaked from a different profile: %+v", row)
		}
	}

	resultB, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{IntegrationProfileID: &profileBID}, dto.PaginationInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListDemandInboxRows profile B: %v", err)
	}
	if resultB.Pagination.TotalCount != 1 {
		t.Fatalf("expected 1 document for profile B, got %d", resultB.Pagination.TotalCount)
	}

	// Assignment filter narrows further: only doc1 (profile A) is assigned.
	resultAssigned, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{Assignment: "assigned"}, dto.PaginationInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListDemandInboxRows assigned filter: %v", err)
	}
	if resultAssigned.Pagination.TotalCount != 1 {
		t.Fatalf("expected 1 assigned document, got %d", resultAssigned.Pagination.TotalCount)
	}
}

func TestListDemandInboxRows_WaveFilter(t *testing.T) {
	gdb := setupDemandInboxTestDB(t)
	_, _, waveID, docIDs := seedDemandInboxFixture(t, gdb)
	c := newDemandInboxTestController(gdb)

	result, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{WaveID: &waveID}, dto.PaginationInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListDemandInboxRows wave filter: %v", err)
	}
	if result.Pagination.TotalCount != 1 || len(result.Rows) != 1 {
		t.Fatalf("expected one document for wave %d, got %+v", waveID, result)
	}
	if result.Rows[0].DemandDocumentID != docIDs[0] || result.Rows[0].AssignedWaveID == nil || *result.Rows[0].AssignedWaveID != waveID {
		t.Fatalf("unexpected wave-filtered row: %+v", result.Rows[0])
	}

	otherWaveID := waveID + 1000
	empty, err := c.ListDemandInboxRows(dto.DemandInboxFilterInput{WaveID: &otherWaveID}, dto.PaginationInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListDemandInboxRows unmatched wave filter: %v", err)
	}
	if empty.Pagination.TotalCount != 0 || len(empty.Rows) != 0 {
		t.Fatalf("expected no documents for wave %d, got %+v", otherWaveID, empty)
	}
}
