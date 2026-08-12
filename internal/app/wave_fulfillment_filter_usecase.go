package app

import (
	"context"
	"sort"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// WaveFulfillmentFilterUseCase implements the server-side filtered/paginated
// fulfillment grid query and the typed/sorted wave list pagination (plan 3.3.2 /
// 5.4). It is declared as a standalone interface — not added to ports.go's
// WaveOverviewQueryUseCase — so it can be constructed and consumed with its full,
// concrete method set from a brand-new controller file without touching
// controller_wave.go or ports.go. See controller_wave_lifecycle.go.
type WaveFulfillmentFilterUseCase interface {
	ListWaveFulfillmentRowsFiltered(ctx context.Context, input dto.WaveFulfillmentFilterInput) (dto.WaveFulfillmentRowsPage, error)
	ListWavesPaginatedTyped(ctx context.Context, input dto.WaveListFilterInput) (dto.WavesPage, error)
}

type waveFulfillmentFilterUseCase struct {
	waveRepo        domain.WaveRepository
	overviewQueryUC WaveOverviewQueryUseCase
}

// NewWaveFulfillmentFilterUseCase constructs a WaveFulfillmentFilterUseCase. It
// reuses WaveOverviewQueryUseCase.ListWaveFulfillmentRows (N+1-fixed by this same
// unit, see wave_overview_query_usecase.go) for row assembly and applies
// filtering/pagination on top, rather than re-querying repositories directly — this
// keeps exactly one code path responsible for turning FulfillmentLines into rows.
func NewWaveFulfillmentFilterUseCase(waveRepo domain.WaveRepository, overviewQueryUC WaveOverviewQueryUseCase) WaveFulfillmentFilterUseCase {
	return &waveFulfillmentFilterUseCase{waveRepo: waveRepo, overviewQueryUC: overviewQueryUC}
}

// waveFilterStringInSet reports whether v is present in set. An empty set matches
// everything (i.e. "no filter on this dimension").
func waveFilterStringInSet(v string, set []string) bool {
	if len(set) == 0 {
		return true
	}
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func matchesFulfillmentKeyword(row dto.WaveFulfillmentRowDTO, keyword string) bool {
	if keyword == "" {
		return true
	}
	k := strings.ToLower(keyword)
	haystacks := []string{row.ParticipantDisplay, row.ProductDisplay, row.DemandSourceSummary, row.LineReason}
	for _, h := range haystacks {
		if strings.Contains(strings.ToLower(h), k) {
			return true
		}
	}
	return false
}

// ListWaveFulfillmentRowsFiltered applies the four state-dim multi-select filters
// (AND across dimensions, OR within a dimension), reviewRequirement, drift status,
// and keyword search, then paginates the result (plan 3.3.2).
func (uc *waveFulfillmentFilterUseCase) ListWaveFulfillmentRowsFiltered(ctx context.Context, input dto.WaveFulfillmentFilterInput) (dto.WaveFulfillmentRowsPage, error) {
	allRows, err := uc.overviewQueryUC.ListWaveFulfillmentRows(ctx, input.WaveID)
	if err != nil {
		return dto.WaveFulfillmentRowsPage{}, err
	}

	filtered := make([]dto.WaveFulfillmentRowDTO, 0, len(allRows))
	for _, row := range allRows {
		if !waveFilterStringInSet(row.AllocationState, input.AllocationStates) {
			continue
		}
		if !waveFilterStringInSet(row.AddressState, input.AddressStates) {
			continue
		}
		if !waveFilterStringInSet(row.SupplierState, input.SupplierStates) {
			continue
		}
		if !waveFilterStringInSet(row.ChannelSyncState, input.ChannelSyncStates) {
			continue
		}
		if !waveFilterStringInSet(row.ReviewRequirement, input.ReviewRequirements) {
			continue
		}
		if !waveFilterStringInSet(row.BasisDriftStatus, input.DriftStatuses) {
			continue
		}
		if !matchesFulfillmentKeyword(row, input.Keyword) {
			continue
		}
		filtered = append(filtered, row)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].FulfillmentLineID < filtered[j].FulfillmentLineID
	})

	pagination := dto.NormalizePagination(input.Pagination)
	total := len(filtered)
	offset := (pagination.Page - 1) * pagination.PageSize
	page := []dto.WaveFulfillmentRowDTO{}
	if offset < total {
		end := offset + pagination.PageSize
		if end > total {
			end = total
		}
		page = filtered[offset:end]
	}

	result := dto.PaginationResult{Page: pagination.Page, PageSize: pagination.PageSize, TotalCount: total}
	result.ComputePages()

	return dto.WaveFulfillmentRowsPage{Items: page, Pagination: result}, nil
}

var waveListSortableFields = map[string]bool{
	"id": true, "name": true, "waveNo": true, "lifecycleStage": true, "createdAt": true, "updatedAt": true,
}

// waveListSortLess compares two waves by the given (already-validated) sort field.
func waveListSortLess(a, b domain.Wave, sortBy string) bool {
	switch sortBy {
	case "name":
		return a.Name < b.Name
	case "waveNo":
		return a.WaveNo < b.WaveNo
	case "lifecycleStage":
		return a.LifecycleStage < b.LifecycleStage
	case "createdAt":
		return a.CreatedAt.Before(b.CreatedAt)
	case "updatedAt":
		return a.UpdatedAt.Before(b.UpdatedAt)
	default:
		return a.ID < b.ID
	}
}

// filterWaves applies the picker/list filters in memory. The wave list is small by
// design (single operator, tens of waves); server-side SQL filtering is not worth
// the extra repository surface (spec §2 follow-ups).
func filterWaves(waves []domain.Wave, input dto.WaveListFilterInput) []domain.Wave {
	out := waves[:0]
	keyword := strings.ToLower(strings.TrimSpace(input.NameKeyword))
	for _, w := range waves {
		if input.LifecycleStage != "" && w.LifecycleStage != input.LifecycleStage {
			continue
		}
		if input.WaveType != "" && w.WaveType != input.WaveType {
			continue
		}
		if keyword != "" &&
			!strings.Contains(strings.ToLower(w.Name), keyword) &&
			!strings.Contains(strings.ToLower(w.WaveNo), keyword) {
			continue
		}
		out = append(out, w)
	}
	return out
}

// ListWavesPaginatedTyped returns a typed, sorted, paginated page of waves. Waves
// are loaded via WaveRepository.List and sorted/paginated in memory rather than
// pushed down to SQL — the plan explicitly notes wave counts are small enough that
// this is acceptable (plan 5.4: "ListWavesPaginated 补类型化 DTO 并真正实现
// SortBy | 或废弃，波次量小可暂缓"). The picker/list filters (lifecycle stage,
// wave type, name/WaveNo keyword) are applied in memory via filterWaves.
func (uc *waveFulfillmentFilterUseCase) ListWavesPaginatedTyped(ctx context.Context, input dto.WaveListFilterInput) (dto.WavesPage, error) {
	waves, err := uc.waveRepo.List(ctx)
	if err != nil {
		return dto.WavesPage{}, err
	}

	waves = filterWaves(waves, input)

	sortBy := input.SortBy
	if !waveListSortableFields[sortBy] {
		sortBy = "id"
	}
	sort.SliceStable(waves, func(i, j int) bool {
		if input.SortDesc {
			return waveListSortLess(waves[j], waves[i], sortBy)
		}
		return waveListSortLess(waves[i], waves[j], sortBy)
	})

	pagination := dto.NormalizePagination(input.PaginationInput)
	total := len(waves)
	offset := (pagination.Page - 1) * pagination.PageSize
	page := []domain.Wave{}
	if offset < total {
		end := offset + pagination.PageSize
		if end > total {
			end = total
		}
		page = waves[offset:end]
	}

	items := make([]dto.WaveDTO, len(page))
	for i := range page {
		items[i] = toWaveDTO(&page[i])
	}

	result := dto.PaginationResult{Page: pagination.Page, PageSize: pagination.PageSize, TotalCount: total}
	result.ComputePages()

	return dto.WavesPage{Items: items, Pagination: result}, nil
}
