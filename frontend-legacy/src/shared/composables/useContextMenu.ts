import { reactive, provide, inject, type InjectionKey } from 'vue'

export interface ContextMenuItem {
  label: string
  key: string
  action: () => void
  divider?: boolean
}

interface MenuState {
  visible: boolean
  x: number
  y: number
  items: ContextMenuItem[]
}

export interface ContextMenuProvider {
  state: MenuState
  show: (x: number, y: number, items: ContextMenuItem[]) => void
  hide: () => void
  register: (key: string, handler: (event: MouseEvent) => ContextMenuItem[]) => () => void
  handleEvent: (event: MouseEvent) => boolean
}

const contextMenuKey: InjectionKey<ContextMenuProvider> = Symbol('contextMenu')

export function useContextMenu() {
  // If injected, return the existing provider (avoids duplicate state)
  const existing = inject(contextMenuKey, null)
  if (existing) return existing

  const state = reactive<MenuState>({
    visible: false,
    x: 0,
    y: 0,
    items: [],
  })

  const handlers = new Map<string, (event: MouseEvent) => ContextMenuItem[]>()

  function show(x: number, y: number, items: ContextMenuItem[]) {
    state.x = x
    state.y = y
    state.items = items
    state.visible = true
  }

  function hide() {
    state.visible = false
  }

  function register(
    key: string,
    handler: (event: MouseEvent) => ContextMenuItem[],
  ) {
    handlers.set(key, handler)
    return () => {
      handlers.delete(key)
    }
  }

  function handleEvent(event: MouseEvent): boolean {
    const target = event.target as HTMLElement | null
    if (!target) return false
    const el = target.closest<HTMLElement>('[data-contextmenu]')
    if (!el) return false
    const key = el.dataset.contextmenu
    if (!key) return false
    const handler = handlers.get(key)
    if (!handler) return false
    const items = handler(event)
    if (items.length === 0) return false
    show(event.clientX, event.clientY, items)
    return true
  }

  const provider: ContextMenuProvider = { state, show, hide, register, handleEvent }
  provide(contextMenuKey, provider)
  return provider
}
