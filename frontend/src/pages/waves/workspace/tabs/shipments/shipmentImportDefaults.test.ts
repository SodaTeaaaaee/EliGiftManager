import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/api/bridge', () => ({
  getDefaultTemplateForProfile: vi.fn(),
}))

import { getDefaultTemplateForProfile } from '@/shared/api/bridge'
import {
  hasHeaderFromMappingRules,
  loadShipmentImportDefaultBinding,
  SUPPLIER_SHIPMENT_IMPORT_DOCUMENT_TYPE,
} from './shipmentImportDefaults'

const getTemplate = vi.mocked(getDefaultTemplateForProfile)

describe('hasHeaderFromMappingRules', () => {
  it('defaults true when mappingRules is missing or empty', () => {
    expect(hasHeaderFromMappingRules(undefined)).toBe(true)
    expect(hasHeaderFromMappingRules(null)).toBe(true)
    expect(hasHeaderFromMappingRules('')).toBe(true)
    expect(hasHeaderFromMappingRules('   ')).toBe(true)
  })

  it('returns false only when hasHeader is explicitly false', () => {
    expect(hasHeaderFromMappingRules('{"hasHeader":false}')).toBe(false)
    expect(hasHeaderFromMappingRules('{"hasHeader":true}')).toBe(true)
  })

  it('defaults true when the key is missing or JSON is invalid', () => {
    expect(hasHeaderFromMappingRules('{"mode":"header"}')).toBe(true)
    expect(hasHeaderFromMappingRules('not-json')).toBe(true)
  })
})

describe('loadShipmentImportDefaultBinding', () => {
  beforeEach(() => {
    getTemplate.mockReset()
  })

  it('loads templateKey and hasHeader from mappingRules', async () => {
    getTemplate.mockResolvedValue({
      templateKey: 'factory-ship',
      mappingRules: '{"hasHeader":false}',
    } as Awaited<ReturnType<typeof getDefaultTemplateForProfile>>)

    await expect(loadShipmentImportDefaultBinding(3)).resolves.toEqual({
      status: 'loaded',
      templateKey: 'factory-ship',
      hasHeader: false,
    })
    expect(getTemplate).toHaveBeenCalledWith(3, SUPPLIER_SHIPMENT_IMPORT_DOCUMENT_TYPE)
  })

  it('defaults hasHeader true when mappingRules is missing or unparseable', async () => {
    getTemplate.mockResolvedValue({
      templateKey: 'factory-ship',
      mappingRules: '{',
    } as Awaited<ReturnType<typeof getDefaultTemplateForProfile>>)

    await expect(loadShipmentImportDefaultBinding(3)).resolves.toEqual({
      status: 'loaded',
      templateKey: 'factory-ship',
      hasHeader: true,
    })
  })

  it('returns missing when the profile has no default template', async () => {
    getTemplate.mockResolvedValue(null)
    await expect(loadShipmentImportDefaultBinding(3)).resolves.toEqual({ status: 'missing' })
  })

  it('returns error when the lookup throws', async () => {
    getTemplate.mockRejectedValue(new Error('network down'))
    await expect(loadShipmentImportDefaultBinding(3)).resolves.toEqual({
      status: 'error',
      message: 'network down',
    })
  })
})
