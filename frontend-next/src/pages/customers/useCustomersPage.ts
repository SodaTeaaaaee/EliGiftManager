/**
 * Shared list/detail state for the customer area (plan §3.6). Two
 * composables:
 *
 * - `useCustomerList()` — server-paginates and filters customer profiles;
 *   identity-platform options come from the dedicated distinct-value call.
 * - `useCustomerDetail(id)` — loads a single profile by id for the unified
 *   detail page, with a `notFound` flag for the "profile deleted / bad id"
 *   case (`customerDetail.notFound` copy).
 */
import { computed, onBeforeMount, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import {
  getCustomerProfile,
  listCustomerIdentityPlatforms,
  listCustomerProfilesPage,
} from '@/shared/api/bridge'
import { registerRefreshTarget } from '@/shared/lib/view-hotkeys'
import type { CustomerProfileDTO } from '@/entities/customer'

export interface PlatformOption {
  label: string
  value: string
}

const CUSTOMER_LIST_KEYWORD_DEBOUNCE_MS = 300
const CUSTOMER_LIST_DEFAULT_PAGE_SIZE = 50

export function useCustomerList() {
  const profiles = ref<CustomerProfileDTO[]>([])
  const loading = ref(false)
  const keyword = ref('')
  const platform = ref('')
  const missingAddressOnly = ref(false)

  // Immediate UI-bound draft value; `keyword` (above) only commits after a
  // debounce so the server isn't queried on every keystroke. Mirrors
  // `FilterBar.vue`'s draft/commit pattern.
  const keywordDraft = ref('')
  let keywordTimer: ReturnType<typeof setTimeout> | undefined

  function onKeywordInput(value: string): void {
    keywordDraft.value = value
    if (keywordTimer !== undefined) clearTimeout(keywordTimer)
    keywordTimer = setTimeout(() => {
      keywordTimer = undefined
      keyword.value = value
    }, CUSTOMER_LIST_KEYWORD_DEBOUNCE_MS)
  }

  let unregisterRefresh: (() => void) | undefined
  onBeforeUnmount(() => {
    if (keywordTimer !== undefined) clearTimeout(keywordTimer)
    unregisterRefresh?.()
  })

  const page = ref(1)
  const pageSize = ref(CUSTOMER_LIST_DEFAULT_PAGE_SIZE)
  const totalCount = ref(0)
  const sortBy = ref<string | null>(null)
  const sortDir = ref<'asc' | 'desc' | null>(null)
  const platformValues = ref<string[]>([])

  async function refresh(): Promise<void> {
    loading.value = true
    try {
      const result = await listCustomerProfilesPage({
        keyword: keyword.value.trim(),
        platform: platform.value,
        missingAddressOnly: missingAddressOnly.value,
        sortBy: sortBy.value ?? undefined,
        sortDir: sortDir.value ?? undefined,
        limit: pageSize.value,
        offset: (page.value - 1) * pageSize.value,
      })
      totalCount.value = result.totalCount
      if ((page.value - 1) * pageSize.value >= result.totalCount && page.value > 1) {
        page.value = Math.max(1, Math.ceil(result.totalCount / pageSize.value))
        await refresh()
        return
      }
      profiles.value = result.items
    } finally {
      loading.value = false
    }
  }

  async function loadPlatformOptions(): Promise<void> {
    platformValues.value = await listCustomerIdentityPlatforms()
  }

  onBeforeMount(() => {
    unregisterRefresh = registerRefreshTarget(refresh)
    void loadPlatformOptions()
  })

  const platformOptions = computed<PlatformOption[]>(() =>
    platformValues.value
      .map((value) => ({ label: value, value }))
  )

  function onPageChange(nextPage: number, nextPageSize: number): void {
    page.value = nextPageSize === pageSize.value ? nextPage : 1
    pageSize.value = nextPageSize
    void refresh()
  }

  function onSort(nextSortBy: string | null, nextSortDir: 'asc' | 'desc' | null): void {
    sortBy.value = nextSortBy
    sortDir.value = nextSortDir
    page.value = 1
    void refresh()
  }

  watch([keyword, platform, missingAddressOnly], () => {
    page.value = 1
    void refresh()
  })

  return {
    profiles,
    loading,
    keyword,
    platform,
    missingAddressOnly,
    platformOptions,
    refresh,
    keywordDraft,
    onKeywordInput,
    page,
    pageSize,
    totalCount,
    onPageChange,
    onSort,
  }
}

export function useCustomerDetail(id: Ref<number | null>) {
  const profile = ref<CustomerProfileDTO | null>(null)
  const loading = ref(false)
  const notFound = ref(false)

  async function refresh(): Promise<void> {
    if (id.value == null) return
    loading.value = true
    notFound.value = false
    try {
      profile.value = await getCustomerProfile(id.value)
    } catch {
      profile.value = null
      notFound.value = true
    } finally {
      loading.value = false
    }
  }

  watch(id, () => void refresh(), { immediate: true })

  return { profile, loading, notFound, refresh }
}
