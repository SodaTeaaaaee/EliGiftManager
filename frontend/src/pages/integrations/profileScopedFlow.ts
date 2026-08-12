export interface ProfileScope {
  profileId: number
  generation: number
}

export interface ProfileLoadScope extends ProfileScope {
  loadSequence: number
}

export function captureProfileScope(
  profileId: number | null,
  generation: number,
): ProfileScope | null {
  return profileId == null ? null : { profileId, generation }
}

export function isProfileScopeActive(
  scope: ProfileScope,
  currentProfileId: number | null,
  currentGeneration: number,
): boolean {
  return scope.profileId === currentProfileId && scope.generation === currentGeneration
}

export function isProfileEntityActive(
  scope: ProfileScope,
  currentProfileId: number | null,
  currentGeneration: number,
  entityProfileId: number,
): boolean {
  return entityProfileId === scope.profileId
    && isProfileScopeActive(scope, currentProfileId, currentGeneration)
}

export function isProfileLoadActive(
  scope: ProfileLoadScope,
  currentProfileId: number | null,
  currentGeneration: number,
  currentLoadSequence: number,
  resultProfileId: number,
): boolean {
  return scope.loadSequence === currentLoadSequence
    && resultProfileId === scope.profileId
    && isProfileScopeActive(scope, currentProfileId, currentGeneration)
}
