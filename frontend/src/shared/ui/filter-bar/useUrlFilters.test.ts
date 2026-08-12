// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, reactive } from 'vue'
import { useUrlFilters, type UseUrlFiltersApi } from './useUrlFilters'
import type { FilterSchema } from './types'

// A reactive stand-in for the live vue-router route: `useRoute()` returns this
// object, so `watch(() => route.query)` tracks the `query` property and fires
// when its identity changes — the same shape the composable relies on.
const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(async () => {}),
  route: { query: {} } as { query: Record<string, unknown> },
}))

// vue-router's useRoute() returns the reactive proxy itself; emulate that by
// re-binding `route` to the proxy AFTER imports are available. Accessing the
// raw target would bypass reactivity and `watch(() => route.query)` would
// never fire.
routerMocks.route = reactive(routerMocks.route)

vi.mock('vue-router', () => ({
  useRoute: () => routerMocks.route,
  useRouter: () => ({ replace: routerMocks.replace }),
}))

const SCHEMA = [
  { key: 'demandKind', type: 'enum-multi', dimension: 'demandKind' },
  { key: 'routingDisposition', type: 'enum-multi', dimension: 'routingDisposition' },
] as const satisfies FilterSchema

const TestHost = defineComponent({
  props: { syncToUrl: { type: Boolean, default: true } },
  setup(props) {
    const filters = useUrlFilters(SCHEMA, { syncToUrl: props.syncToUrl })
    return { filters }
  },
  render: () => null,
})

function mountHost(syncToUrl: boolean) {
  const wrapper = mount(TestHost, { props: { syncToUrl } })
  const filters = (wrapper.vm as unknown as { filters: UseUrlFiltersApi<typeof SCHEMA> }).filters
  return { wrapper, filters }
}

const tick = () => new Promise((resolve) => setTimeout(resolve, 5))

describe('useUrlFilters syncToUrl', () => {
  beforeEach(() => {
    routerMocks.replace.mockReset()
    routerMocks.route.query = {}
  })

  it('syncToUrl=false: filter changes never touch route.query', async () => {
    const { wrapper, filters } = mountHost(false)

    filters.toggleEnumValue('demandKind', 'retail_order')
    expect(filters.state.demandKind).toEqual(['retail_order'])
    await tick()

    expect(routerMocks.replace).not.toHaveBeenCalled()
    expect(routerMocks.route.query).toEqual({})
    wrapper.unmount()
  })

  it('syncToUrl=false: pre-existing route.query is ignored at init (fully local state)', () => {
    routerMocks.route.query = { demandKind: 'retail_order', routingDisposition: 'pending_intake' }
    const { wrapper, filters } = mountHost(false)

    expect(filters.state.demandKind).toEqual([])
    expect(filters.state.routingDisposition).toEqual([])
    wrapper.unmount()
  })

  it('syncToUrl=false: external route.query changes do not update state', async () => {
    const { wrapper, filters } = mountHost(false)

    routerMocks.route.query = { routingDisposition: 'pending_intake' }
    await tick()

    expect(filters.state.routingDisposition).toEqual([])
    wrapper.unmount()
  })

  it('syncToUrl=true (default): filter changes are written to route.query', async () => {
    const { wrapper, filters } = mountHost(true)

    filters.toggleEnumValue('demandKind', 'membership_entitlement')
    await tick()

    expect(routerMocks.replace).toHaveBeenCalled()
    const lastCall = routerMocks.replace.mock.calls.at(-1)?.[0]
    expect(lastCall.query).toEqual({ demandKind: 'membership_entitlement' })
    wrapper.unmount()
  })
})
