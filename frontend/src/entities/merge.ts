export interface MergeProfilesInput {
  sourceProfileId: number
  targetProfileId: number
}

export interface MergeProfilesResult {
  migratedIdentityCount: number
  migratedAddressCount: number
  updatedDemandDocs: number
  updatedParticipants: number
  updatedFulfillmentLines: number
}
