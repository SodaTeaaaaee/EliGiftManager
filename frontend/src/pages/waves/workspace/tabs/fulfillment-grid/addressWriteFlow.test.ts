import { describe, expect, it, vi } from 'vitest'
import type { CustomerAddressDTO } from '@/entities/address'
import {
  bindDefaultAddressesForRows,
  createAndBindInlineAddress,
  inlineAddressDraftIsValid,
} from './addressWriteFlow'

const validDraft = {
  label: ' Home ',
  recipientName: ' Recipient ',
  phone: ' 13800138000 ',
  region: ' Guangdong ',
  detail: ' Road 1 ',
}

describe('wave address write flow', () => {
  it('creates a valid labeled address and binds the returned ID to the line', async () => {
    const createAddress = vi.fn(async () => ({ id: 42 } as CustomerAddressDTO))
    const bindAddressToLine = vi.fn(async () => undefined)

    const completed = await createAndBindInlineAddress({
      writesEnabled: true,
      customerProfileId: 7,
      fulfillmentLineId: 99,
      draft: validDraft,
    }, { createAddress, bindAddressToLine })

    expect(completed).toBe(true)
    expect(createAddress).toHaveBeenCalledWith(expect.objectContaining({
      customerProfileId: 7,
      label: 'Home',
      recipientName: 'Recipient',
      country: 'CN',
      province: 'Guangdong',
      addressLine1: 'Road 1',
    }))
    expect(bindAddressToLine).toHaveBeenCalledWith({ fulfillmentLineId: 99, customerAddressId: 42 })
  })

  it('rejects an empty label before either write API is called', async () => {
    const createAddress = vi.fn(async () => ({ id: 42 } as CustomerAddressDTO))
    const bindAddressToLine = vi.fn(async () => undefined)
    const draft = { ...validDraft, label: '   ' }

    expect(inlineAddressDraftIsValid(draft)).toBe(false)
    expect(await createAndBindInlineAddress({
      writesEnabled: true,
      customerProfileId: 7,
      fulfillmentLineId: 99,
      draft,
    }, { createAddress, bindAddressToLine })).toBe(false)
    expect(createAddress).not.toHaveBeenCalled()
    expect(bindAddressToLine).not.toHaveBeenCalled()
  })

  it('blocks inline and batch write flows before any API read or write when the master gate is off', async () => {
    const createAddress = vi.fn(async () => ({ id: 42 } as CustomerAddressDTO))
    const bindAddressToLine = vi.fn(async () => undefined)
    const listAddressesByProfile = vi.fn(async () => [{ id: 1, isDefault: true }])
    const batchBindAddressToLines = vi.fn(async () => [{ success: true }])

    expect(await createAndBindInlineAddress({
      writesEnabled: false,
      customerProfileId: 7,
      fulfillmentLineId: 99,
      draft: validDraft,
    }, { createAddress, bindAddressToLine })).toBe(false)
    expect(await bindDefaultAddressesForRows(false, [
      { fulfillmentLineId: 99, customerProfileId: 7 },
    ], { listAddressesByProfile, batchBindAddressToLines })).toBeNull()
    expect(createAddress).not.toHaveBeenCalled()
    expect(bindAddressToLine).not.toHaveBeenCalled()
    expect(listAddressesByProfile).not.toHaveBeenCalled()
    expect(batchBindAddressToLines).not.toHaveBeenCalled()
  })

  it('caches default-address lookup per profile and reports unresolved and API failures honestly', async () => {
    const listAddressesByProfile = vi.fn(async (profileId: number) => profileId === 7
      ? [{ id: 70, isDefault: true }]
      : [{ id: 80, isDefault: false }])
    const batchBindAddressToLines = vi.fn(async () => [{ success: true }, { success: false }])

    const outcome = await bindDefaultAddressesForRows(true, [
      { fulfillmentLineId: 1, customerProfileId: 7 },
      { fulfillmentLineId: 2, customerProfileId: 7 },
      { fulfillmentLineId: 3, customerProfileId: 8 },
      { fulfillmentLineId: 4, customerProfileId: null },
    ], { listAddressesByProfile, batchBindAddressToLines })

    expect(listAddressesByProfile).toHaveBeenCalledTimes(2)
    expect(batchBindAddressToLines).toHaveBeenCalledWith([
      { fulfillmentLineId: 1, customerAddressId: 70 },
      { fulfillmentLineId: 2, customerAddressId: 70 },
    ])
    expect(outcome).toEqual({ attempted: true, successCount: 1, failureCount: 3 })
  })
})
