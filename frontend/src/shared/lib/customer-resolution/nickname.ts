import type {
  CustomerNameObservationInput,
  DisplayNameEditEvent,
  DisplayNameEditState,
  DisplayNameMode,
  NicknameEpisodeViewModel,
} from './types'

function observationTime(value: string): number {
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? 0 : parsed
}

function episodeStreamKey(observation: CustomerNameObservationInput): string {
  return [observation.kind, observation.sourceNamespace ?? '', observation.originProfileId ?? ''].join('|')
}

function episodeNameKey(observation: CustomerNameObservationInput): string {
  return (observation.normalizedValue ?? observation.displayValue).trim().toLocaleLowerCase()
}

export function buildNicknameTimeline(
  observations: CustomerNameObservationInput[],
  currentDisplayNameObservationId?: number,
): NicknameEpisodeViewModel[] {
  const chronological = [...observations].sort((left, right) => {
    return observationTime(left.firstSeenAt) - observationTime(right.firstSeenAt) || left.id - right.id
  })

  const episodes: NicknameEpisodeViewModel[] = []
  const latestEpisodeByStream = new Map<string, NicknameEpisodeViewModel>()
  for (const observation of chronological) {
    const streamKey = episodeStreamKey(observation)
    const previous = latestEpisodeByStream.get(streamKey)
    const canExtend = previous != null && episodeNameKey(previous) === episodeNameKey(observation)

    if (canExtend && previous) {
      if (observationTime(observation.lastSeenAt) > observationTime(previous.lastSeenAt)) {
        previous.lastSeenAt = observation.lastSeenAt
      }
      previous.observationCount += observation.observationCount
      previous.observationIds.push(observation.id)
      previous.isCurrentDisplayName ||= observation.id === currentDisplayNameObservationId
      continue
    }

    const episode: NicknameEpisodeViewModel = {
      ...observation,
      observationIds: [observation.id],
      isCurrentDisplayName: observation.id === currentDisplayNameObservationId,
    }
    episodes.push(episode)
    latestEpisodeByStream.set(streamKey, episode)
  }

  return episodes.sort((left, right) => {
    return (
      observationTime(right.lastSeenAt) - observationTime(left.lastSeenAt) ||
      observationTime(right.firstSeenAt) - observationTime(left.firstSeenAt) ||
      right.id - left.id
    )
  })
}

export function createDisplayNameEditState(input: {
  name: string
  mode: DisplayNameMode
  autoName: string
  rowVersion: number
}): DisplayNameEditState {
  return {
    mode: input.mode,
    persistedMode: input.mode,
    draftName: input.name,
    pinnedDraftName: input.name,
    persistedName: input.name,
    autoName: input.autoName,
    rowVersion: input.rowVersion,
    dirty: false,
  }
}

export function reduceDisplayNameEditState(
  state: DisplayNameEditState,
  event: DisplayNameEditEvent,
): DisplayNameEditState {
  switch (event.type) {
    case 'edit_name':
      return {
        ...state,
        mode: 'pinned',
        draftName: event.value,
        pinnedDraftName: event.value,
        dirty: event.value !== state.persistedName || state.persistedMode !== 'pinned',
      }
    case 'select_mode':
      return {
        ...state,
        mode: event.mode,
        draftName: event.mode === 'auto' ? state.autoName : state.pinnedDraftName,
        dirty:
          event.mode !== state.persistedMode ||
          (event.mode === 'auto' ? state.autoName !== state.persistedName : state.pinnedDraftName !== state.persistedName),
      }
    case 'replace_auto_name':
      return {
        ...state,
        autoName: event.value,
        draftName: state.mode === 'auto' ? event.value : state.draftName,
        dirty: state.mode === 'auto' ? event.value !== state.persistedName : state.dirty,
      }
    case 'reset':
      return {
        ...state,
        mode: state.persistedMode,
        draftName: state.persistedName,
        pinnedDraftName: state.persistedName,
        dirty: false,
      }
    case 'saved':
      return {
        mode: event.mode,
        persistedMode: event.mode,
        draftName: event.name,
        pinnedDraftName: event.name,
        persistedName: event.name,
        autoName: state.autoName,
        rowVersion: event.rowVersion,
        dirty: false,
      }
  }
}

export function canSaveDisplayName(state: DisplayNameEditState): boolean {
  return state.dirty && (state.mode === 'auto' || state.draftName.trim().length > 0)
}
