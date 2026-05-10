import { computed, nextTick, ref, watch, type Ref } from 'vue'
import type { TableMode } from '@/shared/model/settings'

export interface StableTableOptions {
  layoutRef: Ref<HTMLElement | null>
  tableRef: Ref<HTMLElement | null>
  footerRef?: Ref<HTMLElement | null>
  rowHeightHint: number
  minPageSize?: number
  headerHeightHint?: number
  footerHeightHint?: number       // fallback when footerRef is null/not provided
  indicatorMinHeight?: number     // minimum height to reserve for the indicator area
}

export function useAdaptiveTable<T>(
  items: Ref<T[]>,
  mode: Ref<TableMode>,
  options: StableTableOptions,
) {
  const minPageSize = options.minPageSize ?? 1
  const viewportHeight = ref(400)
  const viewportWidth = ref(0)
  const measuredHeaderH = ref(options.headerHeightHint ?? 38)
  const currentPage = ref(1)
  const pageGeometryVersion = ref(0)

  const footerHeightHintVal = options.footerHeightHint ?? 32
  const indicatorMinH = options.indicatorMinHeight ?? 0

  function measureFooterHeight(): number {
    if (!options.footerRef?.value) return footerHeightHintVal
    // Use getBoundingClientRect because NPagination uses transform: scale()
    return options.footerRef.value.getBoundingClientRect().height || footerHeightHintVal
  }

  const measuredFooterH = ref(footerHeightHintVal)

  function measureHeaderHeight(): number {
    if (!options.tableRef.value) return measuredHeaderH.value
    const thead = options.tableRef.value.querySelector('.n-data-table-thead')
    return thead instanceof HTMLElement ? thead.offsetHeight : (options.headerHeightHint ?? 38)
  }

  const bodyHeight = computed(() => {
    return Math.max(0, viewportHeight.value - measuredHeaderH.value)
  })

  // Effective body height for page size in paginated mode
  const paginatedBodyHeight = computed(() => {
    return Math.max(0, viewportHeight.value - measuredHeaderH.value - measuredFooterH.value - indicatorMinH)
  })

  const pageSize = computed(() => {
    if (mode.value === 'scroll') return items.value.length
    if (paginatedBodyHeight.value <= 0) return minPageSize
    return Math.max(minPageSize, Math.floor(paginatedBodyHeight.value / options.rowHeightHint))
  })

  const pageCount = computed(() => {
    if (mode.value === 'scroll') return 1
    return Math.max(1, Math.ceil(items.value.length / pageSize.value))
  })

  const renderItems = computed(() => {
    if (mode.value === 'scroll') return items.value
    const start = (currentPage.value - 1) * pageSize.value
    return items.value.slice(start, start + pageSize.value)
  })

  const tableBodyMaxHeight = computed<number | undefined>(() => {
    if (mode.value === 'paginated') return undefined
    return bodyHeight.value
  })

  function clampCurrentPage() {
    if (currentPage.value > pageCount.value) {
      currentPage.value = Math.max(1, pageCount.value)
    }
  }

  function handlePageChange(p: number) {
    currentPage.value = p
  }

  function ensureObserver() {
    if (resizeObserver && options.layoutRef.value) {
      // Re-attempt observe if element now exists but observer isn't watching it
      // ResizeObserver.observe is idempotent per element, so safe to call
      try { resizeObserver.observe(options.layoutRef.value) } catch (_) { /* already observing */ }
    } else if (!resizeObserver && options.layoutRef.value) {
      setupResizeObserver()
    }
  }

  function refreshLayout() {
    if (options.layoutRef.value) {
      const w = options.layoutRef.value.clientWidth
      const h = options.layoutRef.value.clientHeight
      if (w > 0) viewportWidth.value = w
      if (h > 0) viewportHeight.value = h
      measuredHeaderH.value = measureHeaderHeight()
      measuredFooterH.value = measureFooterHeight()
    }
    ensureObserver()
    clampCurrentPage()
  }

  // ResizeObserver on layout container
  let resizeObserver: ResizeObserver | null = null

  function setupResizeObserver() {
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const w = entry.contentRect.width
        const h = entry.contentRect.height
        let geomChanged = false
        if (w > 0 && w !== viewportWidth.value) {
          viewportWidth.value = w
          // Width change may change column set → header may resize
          measuredHeaderH.value = measureHeaderHeight()
          geomChanged = true
        }
        if (h > 0 && h !== viewportHeight.value) {
          viewportHeight.value = h
          measuredFooterH.value = measureFooterHeight()
          geomChanged = true
        }
        if (geomChanged) {
          clampCurrentPage()
          pageGeometryVersion.value++
        }
      }
    })
    if (options.layoutRef.value) {
      resizeObserver.observe(options.layoutRef.value)
    }
  }

  function teardown() {
    resizeObserver?.disconnect()
  }

  async function init() {
    await nextTick()
    refreshLayout()
  }

  // When items length changes, clamp the page
  watch(
    () => items.value.length,
    () => {
      clampCurrentPage()
    },
  )

  watch(mode, () => {
    pageGeometryVersion.value++
    clampCurrentPage()
  })

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
    measuredHeaderH,
    measuredFooterH,
    teardown,
    init,
  }
}
