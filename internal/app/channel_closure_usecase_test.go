package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestPlanChannelClosureReusesExistingPendingJob(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	first, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	if first.Job == nil {
		t.Fatal("expected first plan to create a job")
	}

	second, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if second.Job == nil {
		t.Fatal("expected second plan to return the existing job")
	}
	if first.Job.ID != second.Job.ID {
		t.Errorf("job ID = %d, want reused %d", second.Job.ID, first.Job.ID)
	}
	if len(s.channelSync.jobs) != 1 {
		t.Errorf("expected 1 pending job, got %d", len(s.channelSync.jobs))
	}
	if len(second.Items) != 1 {
		t.Fatalf("expected reused job items, got %d", len(second.Items))
	}
	if second.Items[0].ID == 0 {
		t.Error("reused items should be persisted (non-zero ID)")
	}
}

func TestPlanChannelClosureReusesOldestPendingJobWhenDuplicatesExist(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	first, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	if first.Job == nil {
		t.Fatal("expected first plan to create a job")
	}
	// Simulate a legacy duplicate pending job that predated CreateChannelSyncJob reuse.
	dup := &domain.ChannelSyncJob{
		WaveID:               1,
		IntegrationProfileID: 1,
		Direction:            "push_tracking",
		Status:               "pending",
	}
	if err := s.channelSync.CreateJob(context.Background(), dup); err != nil {
		t.Fatalf("seed duplicate pending job: %v", err)
	}
	if len(s.channelSync.jobs) != 2 {
		t.Fatalf("expected 2 seeded pending jobs, got %d", len(s.channelSync.jobs))
	}

	reused, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if reused.Job == nil || reused.Job.ID != first.Job.ID {
		t.Fatalf("expected oldest pending job %d, got %+v", first.Job.ID, reused.Job)
	}
	if len(s.channelSync.jobs) != 2 {
		t.Errorf("plan must not create a third job, got %d", len(s.channelSync.jobs))
	}
}

func TestPlanChannelClosureCreatesNewJobAfterPreviousJobFinished(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"success", "failed", "partial_success"} {
		status := status
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			s := newClosureTestSetup()
			first, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
			if err != nil {
				t.Fatalf("first plan: %v", err)
			}
			s.channelSync.mu.Lock()
			s.channelSync.jobs[first.Job.ID].Status = status
			s.channelSync.mu.Unlock()

			second, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
			if err != nil {
				t.Fatalf("second plan: %v", err)
			}
			if second.Job == nil {
				t.Fatal("expected a new job after the previous job left pending")
			}
			if second.Job.ID == first.Job.ID {
				t.Errorf("reused job %d despite status %q", first.Job.ID, status)
			}
			if len(s.channelSync.jobs) != 2 {
				t.Errorf("expected 2 jobs after %q, got %d", status, len(s.channelSync.jobs))
			}
		})
	}
}

func TestPlanChannelClosureDoesNotReusePendingJobForOtherProfile(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	if _, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1}); err != nil {
		t.Fatalf("plan profile 1: %v", err)
	}

	s.profile.profiles[2] = &domain.IntegrationProfile{
		ID:                  2,
		ProfileKey:          "other.profile",
		TrackingSyncMode:    "api_push",
		ClosurePolicy:       "close_after_sync",
		AllowsManualClosure: true,
	}
	s.demand.docs[20] = &domain.DemandDocument{
		ID:                   20,
		SourceDocumentNo:     "EXT-ORDER-2",
		IntegrationProfileID: uintPtr(2),
	}
	s.fulfill.lines[2] = &domain.FulfillmentLine{ID: 2, WaveID: 1, DemandDocumentID: uintPtr(20), DemandLineID: uintPtr(200)}
	s.demand.linesByID[200] = &domain.DemandLine{ID: 200, SourceLineNo: 1}
	s.shipment.addLine(domain.ShipmentLine{ID: 2, ShipmentID: 1, FulfillmentLineID: 2})

	result, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 2})
	if err != nil {
		t.Fatalf("plan profile 2: %v", err)
	}
	if result.Job == nil {
		t.Fatal("expected a new job for profile 2")
	}
	if len(s.channelSync.jobs) != 2 {
		t.Errorf("expected 2 jobs (one per profile), got %d", len(s.channelSync.jobs))
	}
}

func TestPlanChannelClosureDocumentExportReusesPendingJob(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].TrackingSyncMode = "document_export"

	first, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	second, err := s.uc.PlanChannelClosure(context.Background(), dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if first.Job == nil || second.Job == nil || first.Job.ID != second.Job.ID {
		t.Fatalf("document_export should reuse pending job: first=%v second=%v", first.Job, second.Job)
	}
	if len(s.channelSync.jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(s.channelSync.jobs))
	}
}

func TestLookupPendingPushTrackingJobSurfacesListError(t *testing.T) {
	t.Parallel()
	repo := &listJobsFailingRepo{err: errors.New("db down")}
	_, _, err := lookupPendingPushTrackingJob(context.Background(), repo, 7, 1)
	if err == nil {
		t.Fatal("expected list error")
	}
	if !strings.Contains(err.Error(), "list channel sync jobs for wave 7") {
		t.Errorf("error = %q, want list channel sync jobs for wave 7", err)
	}
}

func TestLookupPendingPushTrackingJobSurfacesItemListError(t *testing.T) {
	t.Parallel()
	inner := newMockChannelSyncRepo()
	inner.jobs[3] = &domain.ChannelSyncJob{ID: 3, WaveID: 1, IntegrationProfileID: 1, Direction: "push_tracking", Status: "pending"}
	repo := &listItemsFailingRepo{mockChannelSyncRepo: inner, err: errors.New("items missing")}
	_, _, err := lookupPendingPushTrackingJob(context.Background(), repo, 1, 1)
	if err == nil {
		t.Fatal("expected item list error")
	}
	if !strings.Contains(err.Error(), "list items for pending channel sync job 3") {
		t.Errorf("error = %q, want list items for pending channel sync job 3", err)
	}
}

type listJobsFailingRepo struct {
	err error
}

func (r *listJobsFailingRepo) CreateJob(context.Context, *domain.ChannelSyncJob) error {
	return fmt.Errorf("not implemented")
}
func (r *listJobsFailingRepo) FindJobByID(context.Context, uint) (*domain.ChannelSyncJob, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *listJobsFailingRepo) ListJobsByWave(context.Context, uint) ([]domain.ChannelSyncJob, error) {
	return nil, r.err
}
func (r *listJobsFailingRepo) SaveJob(context.Context, *domain.ChannelSyncJob) error {
	return fmt.Errorf("not implemented")
}
func (r *listJobsFailingRepo) CreateItem(context.Context, *domain.ChannelSyncItem) error {
	return fmt.Errorf("not implemented")
}
func (r *listJobsFailingRepo) SaveItem(context.Context, *domain.ChannelSyncItem) error {
	return fmt.Errorf("not implemented")
}
func (r *listJobsFailingRepo) ListItemsByJob(context.Context, uint) ([]domain.ChannelSyncItem, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *listJobsFailingRepo) AtomicCreateChannelSync(context.Context, *domain.ChannelSyncJob, []*domain.ChannelSyncItem, *domain.BasisPinParam) error {
	return fmt.Errorf("not implemented")
}
func (r *listJobsFailingRepo) CountJobsByProfileID(context.Context, uint) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

type listItemsFailingRepo struct {
	*mockChannelSyncRepo
	err error
}

func (r *listItemsFailingRepo) ListItemsByJob(context.Context, uint) ([]domain.ChannelSyncItem, error) {
	return nil, r.err
}
