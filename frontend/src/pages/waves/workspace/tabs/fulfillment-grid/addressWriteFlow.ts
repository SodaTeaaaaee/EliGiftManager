import type { CreateAddressInput, CustomerAddressDTO } from '@/entities/address'

export interface InlineAddressDraft {
  label: string
  recipientName: string
  phone: string
  region: string
  detail: string
}

export interface InlineAddressWriteInput {
  writesEnabled: boolean
  customerProfileId: number | null
  fulfillmentLineId: number
  draft: InlineAddressDraft
}

export interface InlineAddressWriteDependencies {
  createAddress: (input: CreateAddressInput) => Promise<CustomerAddressDTO>
  bindAddressToLine: (input: {
    fulfillmentLineId: number
    customerAddressId: number
  }) => Promise<unknown>
}

export function inlineAddressDraftIsValid(draft: InlineAddressDraft): boolean {
  return draft.label.trim().length > 0
    && draft.recipientName.trim().length > 0
    && draft.phone.trim().length > 0
    && draft.detail.trim().length > 0
}

export function canRunInlineAddressWrite(
  writesEnabled: boolean,
  creating: boolean,
  customerProfileId: number | null,
  draft: InlineAddressDraft,
): boolean {
  return writesEnabled && !creating && customerProfileId != null && inlineAddressDraftIsValid(draft)
}

export function buildInlineAddressCreateInput(
  customerProfileId: number,
  draft: InlineAddressDraft,
): CreateAddressInput | null {
  if (!inlineAddressDraftIsValid(draft)) return null
  return {
    customerProfileId,
    label: draft.label.trim(),
    recipientName: draft.recipientName.trim(),
    phone: draft.phone.trim(),
    country: 'CN',
    province: draft.region.trim(),
    city: '',
    district: '',
    addressLine1: draft.detail.trim(),
    addressLine2: '',
    postalCode: '',
    isDefault: false,
    isTest: false,
    validationStatus: 'unvalidated',
    validationDetail: '',
    extraData: '',
  }
}

export async function createAndBindInlineAddress(
  input: InlineAddressWriteInput,
  dependencies: InlineAddressWriteDependencies,
): Promise<boolean> {
  if (!input.writesEnabled || input.customerProfileId == null) return false
  const createInput = buildInlineAddressCreateInput(input.customerProfileId, input.draft)
  if (!createInput) return false
  const address = await dependencies.createAddress(createInput)
  await dependencies.bindAddressToLine({
    fulfillmentLineId: input.fulfillmentLineId,
    customerAddressId: address.id,
  })
  return true
}

export interface AddressBindingRow {
  fulfillmentLineId: number
  customerProfileId: number | null
}

export interface BatchAddressWriteOutcome {
  attempted: boolean
  successCount: number
  failureCount: number
}

export interface BatchAddressWriteDependencies {
  listAddressesByProfile: (profileId: number) => Promise<Array<Pick<CustomerAddressDTO, 'id' | 'isDefault'>>>
  batchBindAddressToLines: (entries: Array<{
    fulfillmentLineId: number
    customerAddressId: number
  }>) => Promise<Array<{ success: boolean }>>
}

export async function bindDefaultAddressesForRows(
  writesEnabled: boolean,
  selectedRows: readonly AddressBindingRow[],
  dependencies: BatchAddressWriteDependencies,
): Promise<BatchAddressWriteOutcome | null> {
  if (!writesEnabled) return null
  const rows = selectedRows.map((row) => ({ ...row }))
  const rowsWithProfile = rows.filter(
    (row): row is AddressBindingRow & { customerProfileId: number } => row.customerProfileId != null,
  )
  let unresolvedCount = rows.length - rowsWithProfile.length
  const defaultAddressByProfile = new Map<number, number | null>()
  const entries: Array<{ fulfillmentLineId: number; customerAddressId: number }> = []

  for (const row of rowsWithProfile) {
    if (!defaultAddressByProfile.has(row.customerProfileId)) {
      const addresses = await dependencies.listAddressesByProfile(row.customerProfileId)
      const defaultAddress = addresses.find((address) => address.isDefault)
      defaultAddressByProfile.set(row.customerProfileId, defaultAddress?.id ?? null)
    }
    const addressId = defaultAddressByProfile.get(row.customerProfileId) ?? null
    if (addressId == null) {
      unresolvedCount += 1
      continue
    }
    entries.push({ fulfillmentLineId: row.fulfillmentLineId, customerAddressId: addressId })
  }

  if (entries.length === 0) {
    return { attempted: false, successCount: 0, failureCount: rows.length }
  }
  const results = await dependencies.batchBindAddressToLines(entries)
  const successCount = results.filter((result) => result.success).length
  return {
    attempted: true,
    successCount,
    failureCount: results.length - successCount + unresolvedCount,
  }
}
