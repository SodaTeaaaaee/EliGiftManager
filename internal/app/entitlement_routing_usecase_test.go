package app

import (
	"context"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestGetWaveRoutingStats_AcceptedReadyOnlyReadyAndNotRequired(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	uc := NewEntitlementRoutingUseCase(demandRepo, assignmentRepo)

	doc := &domain.DemandDocument{}
	if err := demandRepo.Create(context.Background(), doc); err != nil {
		t.Fatalf("create demand document: %v", err)
	}

	lines := []domain.DemandLine{
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: "ready"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: "not_required"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: "waiting_for_input"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: "partially_collected"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: "waived"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: "expired"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: ""},
		{DemandDocumentID: doc.ID, RoutingDisposition: "accepted", RecipientInputState: "unknown"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "deferred", RecipientInputState: "ready"},
		{DemandDocumentID: doc.ID, RoutingDisposition: "excluded_manual", RecipientInputState: "ready"},
	}
	for i := range lines {
		line := lines[i]
		if err := demandRepo.CreateLine(context.Background(), &line); err != nil {
			t.Fatalf("create demand line: %v", err)
		}
	}
	if err := assignmentRepo.Create(context.Background(), &domain.WaveDemandAssignment{
		WaveID:           1,
		DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	stats, err := uc.GetWaveRoutingStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetWaveRoutingStats: %v", err)
	}
	if stats.TotalLines != 10 {
		t.Errorf("TotalLines = %d, want 10", stats.TotalLines)
	}
	if stats.AcceptedReadyCount != 2 {
		t.Errorf("AcceptedReadyCount = %d, want 2 (ready + not_required only)", stats.AcceptedReadyCount)
	}
	if stats.AcceptedWaitingCount != 1 {
		t.Errorf("AcceptedWaitingCount = %d, want 1", stats.AcceptedWaitingCount)
	}
	if stats.AcceptedPartialCount != 1 {
		t.Errorf("AcceptedPartialCount = %d, want 1", stats.AcceptedPartialCount)
	}
	if stats.DeferredCount != 1 {
		t.Errorf("DeferredCount = %d, want 1", stats.DeferredCount)
	}
	if stats.ExcludedManualCount != 1 {
		t.Errorf("ExcludedManualCount = %d, want 1", stats.ExcludedManualCount)
	}
}

func TestUpdateDemandLineRouting_RejectsAcceptedToExcludedAfterAssignment(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	uc := NewEntitlementRoutingUseCase(demandRepo, assignmentRepo)

	doc := &domain.DemandDocument{}
	if err := demandRepo.Create(context.Background(), doc); err != nil {
		t.Fatalf("create demand document: %v", err)
	}
	line := &domain.DemandLine{
		DemandDocumentID:    doc.ID,
		RoutingDisposition:  string(domain.RoutingDispositionAccepted),
		RecipientInputState: string(domain.RecipientInputStateReady),
	}
	if err := demandRepo.CreateLine(context.Background(), line); err != nil {
		t.Fatalf("create demand line: %v", err)
	}
	if err := assignmentRepo.Create(context.Background(), &domain.WaveDemandAssignment{
		WaveID:           1,
		DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	err := uc.UpdateDemandLineRouting(context.Background(), dto.UpdateDemandLineRoutingInput{
		DemandLineID:        line.ID,
		RoutingDisposition:  string(domain.RoutingDispositionExcludedManual),
		RecipientInputState: string(domain.RecipientInputStateReady),
	})
	if err == nil {
		t.Fatal("expected error excluding an assigned accepted line, got nil")
	}
	if !strings.Contains(err.Error(), "assigned") {
		t.Errorf("error %q should mention assignment", err.Error())
	}

	stored, findErr := demandRepo.FindLineByID(context.Background(), line.ID)
	if findErr != nil {
		t.Fatalf("FindLineByID: %v", findErr)
	}
	if stored.RoutingDisposition != string(domain.RoutingDispositionAccepted) {
		t.Errorf("RoutingDisposition = %q, want accepted (unchanged)", stored.RoutingDisposition)
	}
}

func TestUpdateDemandLineRouting_AllowsAcceptedToExcludedWhenUnassigned(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	uc := NewEntitlementRoutingUseCase(demandRepo, assignmentRepo)

	doc := &domain.DemandDocument{}
	if err := demandRepo.Create(context.Background(), doc); err != nil {
		t.Fatalf("create demand document: %v", err)
	}
	line := &domain.DemandLine{
		DemandDocumentID:    doc.ID,
		RoutingDisposition:  string(domain.RoutingDispositionAccepted),
		RecipientInputState: string(domain.RecipientInputStateReady),
	}
	if err := demandRepo.CreateLine(context.Background(), line); err != nil {
		t.Fatalf("create demand line: %v", err)
	}

	err := uc.UpdateDemandLineRouting(context.Background(), dto.UpdateDemandLineRoutingInput{
		DemandLineID:        line.ID,
		RoutingDisposition:  string(domain.RoutingDispositionExcludedManual),
		RecipientInputState: string(domain.RecipientInputStateReady),
	})
	if err != nil {
		t.Fatalf("expected unassigned accepted line to be excludable, got %v", err)
	}

	stored, findErr := demandRepo.FindLineByID(context.Background(), line.ID)
	if findErr != nil {
		t.Fatalf("FindLineByID: %v", findErr)
	}
	if stored.RoutingDisposition != string(domain.RoutingDispositionExcludedManual) {
		t.Errorf("RoutingDisposition = %q, want excluded_manual", stored.RoutingDisposition)
	}
}
