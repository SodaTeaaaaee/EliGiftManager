// Bridge health: single reactive source of truth for whether the Wails
// runtime is reachable. Populated exclusively by the bridge's runtime-
// availability guard (see shared/api/bridge.ts) — no other module should
// call markBridgeSeen/markBridgeMissing directly.

import { computed, type ComputedRef, ref } from "vue";

export type BridgeState = "unknown" | "available" | "unavailable";

const state = ref<BridgeState>("unknown");

/** Mark the bridge as reachable. Called from the bridge's runtime guard. */
export function markBridgeSeen(): void {
  state.value = "available";
}

/** Mark the bridge as unreachable. Called from the bridge's runtime guard. */
export function markBridgeMissing(): void {
  state.value = "unavailable";
}

/** Reactive bridge connectivity state, for the global disconnected banner. */
export const bridgeState = computed(() => state.value);

export function useBridgeHealth(): { bridgeState: ComputedRef<BridgeState> } {
  return { bridgeState };
}
