import { computed, type Ref, ref } from "vue";
import {
  type SortDescriptor,
  type SortOrder,
  stableSortRows,
} from "@/shared/lib/table/stableSortRows";

export type { SortDescriptor, SortOrder };

export interface TableSortState {
  columnKey: string | null;
  order: SortOrder;
}

export function nextSortOrderAscFirst(order: SortOrder): SortOrder {
  return order === false ? "ascend" : order === "ascend" ? "descend" : false;
}

export function useTableSort<T>(
  items: Ref<T[]>,
  descriptors: SortDescriptor<T>[],
  defaultColumnKey?: string,
  defaultOrder: SortOrder = false,
) {
  const sortState = ref<TableSortState>({
    columnKey: defaultColumnKey ?? null,
    order: defaultOrder,
  });

  const activeDescriptor = computed(() => {
    if (!sortState.value.columnKey) return null;
    return descriptors.find((d) => d.key === sortState.value.columnKey) ?? null;
  });

  const sortedItems = computed(() => {
    return stableSortRows(
      items.value,
      activeDescriptor.value,
      sortState.value.order,
    );
  });

  function toggleSort(columnKey: string) {
    const current = sortState.value;
    if (current.columnKey === columnKey) {
      // Cycle: false → ascend → descend → false
      if (current.order === false) {
        sortState.value = { columnKey, order: "ascend" };
      } else if (current.order === "ascend") {
        sortState.value = { columnKey, order: "descend" };
      } else {
        sortState.value = { columnKey: null, order: false };
      }
    } else {
      sortState.value = { columnKey, order: "ascend" };
    }
  }

  function applySorter(
    sorter: { columnKey: string | null; order: SortOrder } | null,
  ) {
    if (!sorter || !sorter.columnKey || sorter.order === false) {
      sortState.value = { columnKey: null, order: false };
    } else {
      sortState.value = { columnKey: sorter.columnKey, order: sorter.order };
    }
  }

  function applyNaiveSorterEvent(
    sorter: { columnKey: string | null; order: SortOrder } | null,
  ) {
    const clickedColumnKey = sorter?.columnKey ?? null
    if (!clickedColumnKey) {
      sortState.value = { columnKey: null, order: false }
      return
    }

    const current = sortState.value
    if (current.columnKey !== clickedColumnKey) {
      // New column clicked → always start with ascend
      sortState.value = { columnKey: clickedColumnKey, order: 'ascend' }
      return
    }

    // Same column — cycle: ascend → descend → false
    if (current.order === 'ascend') {
      sortState.value = { columnKey: clickedColumnKey, order: 'descend' }
      return
    }

    if (current.order === 'descend') {
      sortState.value = { columnKey: null, order: false }
      return
    }

    // Fallback (should not reach here, but safety)
    sortState.value = { columnKey: clickedColumnKey, order: 'ascend' }
  }

  function clearSort() {
    sortState.value = { columnKey: null, order: false };
  }

  return {
    sortState,
    activeDescriptor,
    sortedItems,
    toggleSort,
    applySorter,
    applyNaiveSorterEvent,
    clearSort,
  };
}
