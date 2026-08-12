package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type MergeGraphService struct{ store domain.MergeExecutionStore }

func NewMergeGraphService(store domain.MergeExecutionStore) *MergeGraphService {
	return &MergeGraphService{store: store}
}

func (s *MergeGraphService) ValidateExecution(ctx context.Context, source, target *domain.CustomerProfile) (*uint, []string, error) {
	blockers := make([]string, 0)
	if !activeMergeProfile(source) || source.MergedIntoProfileID != nil {
		blockers = append(blockers, "source_not_active_root")
	}
	if !activeMergeProfile(target) || target.MergedIntoProfileID != nil {
		blockers = append(blockers, "target_not_active_root")
	}
	visited := map[uint]struct{}{}
	current := target
	for current != nil && current.MergedIntoProfileID != nil {
		if *current.MergedIntoProfileID == source.ID {
			blockers = append(blockers, "merge_graph_cycle")
			break
		}
		if _, ok := visited[current.ID]; ok {
			blockers = append(blockers, "merge_graph_corrupt_cycle")
			break
		}
		visited[current.ID] = struct{}{}
		next, err := s.store.FindProfileForMerge(ctx, *current.MergedIntoProfileID, true)
		if err != nil {
			return nil, nil, fmt.Errorf("follow merge graph from profile %d: %w", current.ID, err)
		}
		current = next
	}
	records, err := s.store.ListActiveMergeRecords(ctx, []uint{source.ID, target.ID})
	if err != nil {
		return nil, nil, fmt.Errorf("list active merge graph edges: %w", err)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].ID > records[j].ID
		}
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	var dependency *uint
	for i := range records {
		if records[i].SourceProfileID == target.ID {
			blockers = append(blockers, "target_has_outgoing_merge")
		}
		if dependency == nil && records[i].TargetProfileID == target.ID {
			id := records[i].ID
			dependency = &id
		}
	}
	return dependency, compactMergeCodes(blockers), nil
}

func activeMergeProfile(profile *domain.CustomerProfile) bool {
	return profile != nil && (profile.Status == "" || profile.Status == domain.CustomerProfileStatusActive)
}

func compactMergeCodes(values []string) []string {
	if len(values) < 2 {
		return values
	}
	sort.Strings(values)
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
