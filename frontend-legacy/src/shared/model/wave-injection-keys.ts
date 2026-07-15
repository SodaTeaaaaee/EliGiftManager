import type { InjectionKey, Ref } from 'vue'
import type { dto } from '@/../wailsjs/go/models'

/** Provided by WaveWorkspaceLayout — increments on every successful undo/redo. */
export const waveRefreshKey: InjectionKey<Ref<number>> = Symbol('waveRefreshKey')

/** Provided by WaveWorkspaceLayout — the current workspace snapshot. */
export const waveWorkspaceSnapshotKey: InjectionKey<Ref<dto.WaveWorkspaceSnapshotDTO | null>> =
  Symbol('waveWorkspaceSnapshot')
