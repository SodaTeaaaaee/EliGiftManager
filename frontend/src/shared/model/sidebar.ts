import { defineStore } from "pinia";
import { ref } from "vue";

const APP_COLLAPSED_KEY = "eligiftmanager:sidebar:app-collapsed";
const WAVE_COLLAPSED_KEY = "eligiftmanager:sidebar:wave-collapsed";

function readBool(key: string): boolean {
  if (typeof window === "undefined") return false;
  return window.localStorage.getItem(key) === "true";
}

function writeBool(key: string, value: boolean): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(key, String(value));
}

/**
 * Persisted collapse state for the two sidebars:
 * - `appCollapsed`  — the global navigation sidebar (AppLayout)
 * - `waveCollapsed` — the wave workspace sidebar (WaveWorkspaceLayout)
 */
export const useSidebarStore = defineStore("sidebar", () => {
  const appCollapsed = ref(readBool(APP_COLLAPSED_KEY));
  const waveCollapsed = ref(readBool(WAVE_COLLAPSED_KEY));

  function setAppCollapsed(value: boolean) {
    appCollapsed.value = value;
    writeBool(APP_COLLAPSED_KEY, value);
  }

  function setWaveCollapsed(value: boolean) {
    waveCollapsed.value = value;
    writeBool(WAVE_COLLAPSED_KEY, value);
  }

  return {
    appCollapsed,
    waveCollapsed,
    setAppCollapsed,
    setWaveCollapsed,
  };
});
