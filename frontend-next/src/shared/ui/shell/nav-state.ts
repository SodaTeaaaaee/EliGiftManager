/**
 * Persisted collapse state for `SideNav`'s icon-rail mode. Deliberately a
 * tiny, single-purpose Pinia store (mirrors the persistence pattern in
 * `shared/theme/theme.ts`) rather than component-local `ref` + `localStorage`
 * boilerplate, so any future surface (e.g. a "collapse sidebar" menu command)
 * can read/toggle the same state without prop-drilling through `AppShell`.
 */
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const STORAGE_KEY = 'eligiftmanager:sidenav-collapsed'

function readStorage(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    // localStorage can throw in locked-down / privacy-mode contexts — default
    // to expanded rather than crash the shell.
    return false
  }
}

function writeStorage(value: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, value ? '1' : '0')
  } catch {
    // best-effort persistence only
  }
}

export const useSideNavStore = defineStore('shell-side-nav', () => {
  const collapsed = ref<boolean>(readStorage())

  function toggle(): void {
    collapsed.value = !collapsed.value
  }

  function setCollapsed(next: boolean): void {
    collapsed.value = next
  }

  watch(collapsed, writeStorage)

  return { collapsed, toggle, setCollapsed }
})
