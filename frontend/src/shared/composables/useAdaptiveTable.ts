import { computed, nextTick, type Ref, ref, watch } from "vue";
import type { TableMode } from "@/shared/model/settings";
import {
  packMeasuredRows,
  type PageRange,
} from "@/shared/lib/table/packMeasuredRows";

export interface StableTableOptions {
  layoutRef: Ref<HTMLElement | null>;
  tableRef: Ref<HTMLElement | null>;
  paginationRef?: Ref<HTMLElement | null>;
  rowHeightHint: number | ((viewportWidth: number) => number);
  minPageSize?: number;
  headerHeightHint?: number;
  footerHeightHint?: number; // fallback when paginationRef is null/not provided
  contentSignature?: () => string | number;
}

export function useAdaptiveTable<T>(
  items: Ref<T[]>,
  mode: Ref<TableMode>,
  options: StableTableOptions,
) {
  const minPageSize = options.minPageSize ?? 1;
  const viewportHeight = ref(400);
  const viewportWidth = ref(0);
  const measuredHeaderH = ref(options.headerHeightHint ?? 38);
  const currentPage = ref(1);
  const pageGeometryVersion = ref(0);
  const measurementInvalidationVersion = ref(0);
  const measurementRequestId = ref(0);

  const footerHeightHintVal = options.footerHeightHint ?? 32;

  const resolvedRowHeightHint = computed(() => {
    const h = options.rowHeightHint;
    return typeof h === "function" ? h(viewportWidth.value) : h;
  });

  const measuredRowHeights = ref<number[]>([]);

  function invalidateMeasuredRows() {
    measuredRowHeights.value = [];
  }

  const pageRanges = computed<PageRange[]>(() => {
    if (mode.value === "scroll") {
      return [{ start: 0, end: items.value.length - 1 }];
    }
    const h = measuredRowHeights.value;
    if (h.length === 0) {
      // Fallback: use rowHeightHint to estimate until real measurements arrive
      if (paginatedContentHeight.value <= 0) {
        return [{ start: 0, end: items.value.length - 1 }];
      }
      const estPageSize = Math.max(
        minPageSize,
        Math.floor(paginatedContentHeight.value / resolvedRowHeightHint.value),
      );
      const ranges: PageRange[] = [];
      for (let i = 0; i < items.value.length; i += estPageSize) {
        ranges.push({
          start: i,
          end: Math.min(i + estPageSize - 1, items.value.length - 1),
        });
      }
      return ranges;
    }
    return packMeasuredRows(h, paginatedContentHeight.value, 4);
  });

  function measureFooterHeight(): number {
    if (!options.paginationRef?.value) return footerHeightHintVal;
    // Use getBoundingClientRect because NPagination uses transform: scale()
    return options.paginationRef.value.getBoundingClientRect().height ||
      footerHeightHintVal;
  }

  const measuredFooterH = ref(footerHeightHintVal);

  function measureHeaderHeight(): number {
    if (!options.tableRef.value) return measuredHeaderH.value;
    const thead = options.tableRef.value.querySelector(".n-data-table-thead");
    return thead instanceof HTMLElement
      ? thead.offsetHeight
      : (options.headerHeightHint ?? 38);
  }

  const bodyHeight = computed(() => {
    return Math.max(0, viewportHeight.value - measuredHeaderH.value);
  });

  // Effective body height for page size in paginated mode
  const paginatedContentHeight = computed(() => {
    return Math.max(0, viewportHeight.value - measuredHeaderH.value);
  });

  const pageSize = computed(() => {
    const range = pageRanges.value[currentPage.value - 1];
    if (!range) return 0;
    return range.end - range.start + 1;
  });

  const pageCount = computed(() => {
    return pageRanges.value.length || 1;
  });

  const renderItems = computed(() => {
    if (mode.value === "scroll") return items.value;
    const range = pageRanges.value[currentPage.value - 1];
    if (!range) return items.value;
    return items.value.slice(range.start, range.end + 1);
  });

  const tableBodyMaxHeight = computed<number | undefined>(() => {
    if (mode.value === "paginated") return undefined;
    return bodyHeight.value;
  });

  function applyMeasuredRows(
    rowHeights: number[],
    headerHeight: number,
    requestId?: number,
  ): boolean {
    if (requestId != null && requestId !== measurementRequestId.value) {
      return false;
    }
    if (rowHeights.length !== items.value.length) return false;
    measuredRowHeights.value = rowHeights;
    if (headerHeight > 0) {
      measuredHeaderH.value = headerHeight;
    }
    clampCurrentPage();
    return true;
  }

  function schedulePostPaintRefresh() {
    nextTick().then(() => {
      requestAnimationFrame(() => {
        refreshLayout();
      });
    });
  }

  async function init() {
    await nextTick();
    requestAnimationFrame(() => {
      refreshLayout();
    });
  }

  function ensureObserver() {
    if (resizeObserver && options.layoutRef.value) {
      // Re-attempt observe if element now exists but observer isn't watching it
      // ResizeObserver.observe is idempotent per element, so safe to call
      try {
        resizeObserver.observe(options.layoutRef.value);
      } catch (_) { /* already observing */ }
    } else if (!resizeObserver && options.layoutRef.value) {
      setupResizeObserver();
    }
  }

  function refreshLayout() {
    if (options.layoutRef.value) {
      const w = options.layoutRef.value.clientWidth;
      const h = options.layoutRef.value.clientHeight;
      if (w > 0) viewportWidth.value = w;
      if (h > 0) viewportHeight.value = h;
      measuredHeaderH.value = measureHeaderHeight();
      measuredFooterH.value = measureFooterHeight();
    }
    ensureObserver();
    clampCurrentPage();
  }

  // ResizeObserver on layout container
  let resizeObserver: ResizeObserver | null = null;

  function setupResizeObserver() {
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const w = entry.contentRect.width;
        const h = entry.contentRect.height;
        let geomChanged = false;
        if (w > 0 && w !== viewportWidth.value) {
          viewportWidth.value = w;
          // Width change may change column set → header may resize
          measuredHeaderH.value = measureHeaderHeight();
          measuredFooterH.value = measureFooterHeight();
          geomChanged = true;
        }
        if (h > 0 && h !== viewportHeight.value) {
          viewportHeight.value = h;
          measuredFooterH.value = measureFooterHeight();
          geomChanged = true;
        }
        if (geomChanged) {
          clampCurrentPage();
          pageGeometryVersion.value++;
          measurementInvalidationVersion.value++;
        }
      }
    });
    if (options.layoutRef.value) {
      resizeObserver.observe(options.layoutRef.value);
    }
  }

  function teardown() {
    resizeObserver?.disconnect();
  }

  function clampCurrentPage() {
    if (currentPage.value > pageCount.value) {
      currentPage.value = Math.max(1, pageCount.value);
    }
  }

  function handlePageChange(p: number) {
    currentPage.value = p;
  }

  // When items length changes, clamp the page
  watch(
    () => items.value.length,
    () => {
      invalidateMeasuredRows();
      clampCurrentPage();
    },
  );

  watch(mode, () => {
    pageGeometryVersion.value++;
    measurementInvalidationVersion.value++;
    clampCurrentPage();
  });

  if (options.contentSignature) {
    watch(options.contentSignature, () => {
      requestRemeasure();
      currentPage.value = 1;
    });
  }

  function requestRemeasure() {
    invalidateMeasuredRows();
    measurementRequestId.value++;
    measurementInvalidationVersion.value++;
  }

  return {
    renderItems,
    tableBodyMaxHeight,
    viewportWidth,
    pageSize,
    currentPage,
    pageCount,
    handlePageChange,
    refreshLayout,
    clampCurrentPage,
    pageGeometryVersion,
    measurementInvalidationVersion,
    measurementRequestId,
    requestRemeasure,
    measuredHeaderH,
    teardown,
    init,
    invalidateMeasuredRows,
    pageRanges,
    applyMeasuredRows,
    schedulePostPaintRefresh,
    measuredRowHeights,
  };
}
