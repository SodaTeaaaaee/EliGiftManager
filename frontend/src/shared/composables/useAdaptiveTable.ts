import { computed, nextTick, type Ref, ref } from "vue";
import { useScrollMode } from "@/shared/model/settings";

export interface AdaptiveTableOptions {
  tableParentRef: Ref<HTMLElement | null>;
  tableWrapperRef: Ref<HTMLElement | null>;
  paginationRef: Ref<HTMLElement | null>;
  indicatorRef: Ref<HTMLElement | null>;
}

function packByHeights(
  heights: number[],
  availableH: number,
  headerH: number,
): Array<{ start: number; end: number }> {
  const pages: Array<{ start: number; end: number }> = [];
  if (heights.length === 0) return pages;
  const bodyH = availableH - headerH;
  if (bodyH <= 0) {
    for (let i = 0; i < heights.length; i++) pages.push({ start: i, end: i });
    return pages;
  }
  let pageStart = 0;
  let used = 0;
  for (let i = 0; i < heights.length; i++) {
    if (used + heights[i] > bodyH && i > pageStart) {
      pages.push({ start: pageStart, end: i - 1 });
      pageStart = i;
      used = heights[i];
    } else {
      used += heights[i];
    }
  }
  pages.push({ start: pageStart, end: heights.length - 1 });
  return pages;
}

export function useAdaptiveTable<T>(
  items: Ref<T[]>,
  opts: AdaptiveTableOptions,
) {
  const scrollMode = useScrollMode();

  // ── heights ──
  const headerH = ref(38);
  const paginationH = ref(32);
  const availableH = ref(400);
  const currentPage = ref(1);
  const lastW = ref(0);

  const needsMeasure = ref(true);
  const measuredHeights = ref<number[]>([]);

  const pages = computed(() =>
    packByHeights(
      measuredHeights.value,
      availableH.value - paginationH.value * 2 - 12,
      headerH.value,
    )
  );

  const totalPages = computed(() => pages.value.length || 1);

  const visibleItems = computed(() => {
    if (scrollMode.value || needsMeasure.value) return items.value;
    const page = pages.value[currentPage.value - 1];
    if (!page) return items.value;
    return items.value.slice(page.start, page.end + 1);
  });

  function handlePageChange(p: number) {
    currentPage.value = p;
  }

  // ── measurement helpers ──
  function measureHeaderHeight(wrapper: HTMLElement | null): number {
    if (!wrapper) return 38;
    const thead = wrapper.querySelector(".n-data-table-thead");
    return thead instanceof HTMLElement ? thead.offsetHeight : 40;
  }

  function measurePaginationHeight(el: HTMLElement | null): number {
    return el ? el.offsetHeight : 32;
  }

  async function remeasure() {
    needsMeasure.value = true;
    await nextTick();
    const trs = opts.tableWrapperRef.value?.querySelectorAll("tbody tr");
    if (trs && trs.length > 0) {
      measuredHeights.value = Array.from(trs).map((tr) =>
        (tr as HTMLElement).offsetHeight
      );
    }
    needsMeasure.value = false;
    if (currentPage.value > pages.value.length) currentPage.value = 1;
  }

  // ── observers ──
  let resizeObserver: ResizeObserver | null = null;
  let indicatorObserver: ResizeObserver | null = null;

  function setupResizeObserver() {
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.target === opts.tableParentRef.value) {
          const w = entry.contentRect.width;
          const h = entry.contentRect.height;
          if (h <= 0) continue;
          if (w !== lastW.value) {
            lastW.value = w;
            remeasure();
          }
          if (h !== availableH.value) {
            availableH.value = h;
            currentPage.value = 1;
          }
        }
      }
    });
    if (opts.tableParentRef.value) {
      resizeObserver.observe(opts.tableParentRef.value);
    }
  }

  function setupIndicatorObserver() {
    indicatorObserver?.disconnect();
    if (opts.indicatorRef.value) {
      indicatorObserver = new ResizeObserver((entries) => {
        for (const entry of entries) {
          indicatorW.value = entry.contentRect.width;
          indicatorH.value = entry.contentRect.height;
        }
      });
      indicatorObserver.observe(opts.indicatorRef.value);
    }
  }

  function teardown() {
    resizeObserver?.disconnect();
    indicatorObserver?.disconnect();
  }

  // ── indicator dimension refs (exposed so caller can bind template) ──
  const indicatorW = ref(0);
  const indicatorH = ref(0);

  const indicatorFontSize = computed(() => {
    const h = indicatorH.value;
    if (h < 16) return 12;
    return Math.min(Math.floor(h * 0.95), 800);
  });

  function makeIndicatorContent(direction: "left" | "right") {
    return computed(() => {
      const current = currentPage.value;
      const total = totalPages.value;
      if (total <= 1 || scrollMode.value) return "";
      const w = indicatorW.value;
      const size = indicatorFontSize.value;
      const charW = Math.max(size * 0.6, 6);
      const count = Math.max(2, Math.floor(w / charW / 2) * 2);
      const half = count / 2;
      if (direction === "left") {
        if (current === 1) return "";
        return "<".repeat(current === total ? count : half);
      }
      if (current === total) return "";
      return ">".repeat(current === 1 ? count : half);
    });
  }

  // ── init helper ──
  async function init() {
    await nextTick();
    headerH.value = measureHeaderHeight(opts.tableWrapperRef.value);
    paginationH.value = measurePaginationHeight(opts.paginationRef.value);
    if (opts.tableParentRef.value) {
      const h = opts.tableParentRef.value.clientHeight;
      if (h > 0) availableH.value = h;
      lastW.value = opts.tableParentRef.value.clientWidth;
    }
    await remeasure();
    setupResizeObserver();
    setupIndicatorObserver();
  }

  return {
    // state
    headerH,
    paginationH,
    availableH,
    currentPage,
    totalPages,
    visibleItems,
    scrollMode,
    lastW,
    // indicator
    indicatorFontSize,
    indicatorLeft: makeIndicatorContent("left"),
    indicatorRight: makeIndicatorContent("right"),
    // actions
    handlePageChange,
    remeasure,
    setupResizeObserver,
    setupIndicatorObserver,
    teardown,
    init,
  };
}
