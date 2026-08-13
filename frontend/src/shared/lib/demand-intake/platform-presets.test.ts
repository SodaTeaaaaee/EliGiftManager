import { describe, expect, test } from 'vitest'
import {
  mappingFromPresetForDocumentType,
  PLATFORM_PRESETS,
  type PlatformPreset,
} from './platform-presets'

function bilibiliPreset() {
  const preset = PLATFORM_PRESETS.find((item) => item.key === 'bilibili')
  expect(preset).toBeDefined()
  return preset!
}

describe('PLATFORM_PRESETS bilibili', () => {
  test('includes a preset whose key is bilibili', () => {
    expect(bilibiliPreset().key).toBe('bilibili')
  })

  test('maps import_sales_order buyer nickname to source_customer_ref and display_name', () => {
    const mapping = mappingFromPresetForDocumentType(bilibiliPreset(), 'import_sales_order')
    expect(mapping).not.toBeNull()
    expect(mapping!.columns['document.source_customer_ref']).toBe('买家昵称')
    expect(mapping!.columns['document.display_name']).toBe('买家昵称')
  })

  test('keeps the rest of the import_sales_order retail headers', () => {
    const mapping = mappingFromPresetForDocumentType(bilibiliPreset(), 'import_sales_order')
    expect(mapping).not.toBeNull()
    expect(mapping!.columns['line.external_title']).toBe('商品名称')
    expect(mapping!.columns['line.requested_quantity']).toBe('数量')
    expect(mapping!.columns['recipient.name']).toBe('收货人姓名')
    expect(mapping!.columns['recipient.phone']).toBe('联系电话')
    expect(mapping!.columns['recipient.address_line1']).toBe('收货地址')
    expect(mapping!.columns['document.source_document_no']).toBe('订单号')
    expect(mapping!.columnOrder).toEqual([
      'line.external_title',
      'line.requested_quantity',
      'recipient.name',
      'recipient.phone',
      'recipient.address_line1',
      'document.source_document_no',
      'document.source_customer_ref',
      'document.display_name',
    ])
  })

  test('defaults import_entitlement recipient_input_state to not_required', () => {
    const mapping = mappingFromPresetForDocumentType(bilibiliPreset(), 'import_entitlement')
    expect(mapping).not.toBeNull()
    expect(mapping!.defaults['line.recipient_input_state']).toBe('not_required')
  })

  test('uses positional entitlement columns for level, uid, and display name', () => {
    const mapping = mappingFromPresetForDocumentType(bilibiliPreset(), 'import_entitlement')
    expect(mapping).not.toBeNull()
    expect(mapping!.positions?.['line.gift_level_snapshot']).toBe(0)
    expect(mapping!.positions?.['document.source_customer_ref']).toBe(1)
    expect(mapping!.positions?.['document.display_name']).toBe(2)
  })

  test('returns null for a document type a single-kind preset does not seed', () => {
    const patreon = PLATFORM_PRESETS.find((item) => item.key === 'patreon')
    expect(patreon).toBeDefined()
    expect(mappingFromPresetForDocumentType(patreon!, 'import_sales_order')).toBeNull()
  })
})

function leftoverPreset(overrides: Partial<PlatformPreset>): PlatformPreset {
  return {
    key: 'leftover',
    labelKey: '',
    descKey: '',
    sourceChannel: '',
    sourceSurface: '',
    defaultColumns: { 'line.external_title': 'Title' },
    defaultCapabilities: {},
    ...overrides,
  }
}

describe('mappingFromPresetForDocumentType leftover demandKind', () => {
  test('empty leftover demandKind does not map onto entitlement', () => {
    const preset = leftoverPreset({ demandKind: '' })
    expect(mappingFromPresetForDocumentType(preset, 'import_entitlement')).toBeNull()
    expect(mappingFromPresetForDocumentType(preset, 'import_sales_order')).toBeNull()
  })

  test('explicit membership_entitlement kind still maps onto entitlement', () => {
    const preset = leftoverPreset({ demandKind: 'membership_entitlement' })
    const mapping = mappingFromPresetForDocumentType(preset, 'import_entitlement')
    expect(mapping).not.toBeNull()
    expect(mapping!.columns['line.external_title']).toBe('Title')
    expect(mappingFromPresetForDocumentType(preset, 'import_sales_order')).toBeNull()
  })

  test('explicit documentType still maps onto entitlement when leftover demandKind is empty', () => {
    const preset = leftoverPreset({
      demandKind: '',
      defaultColumns: undefined,
      mappingsByDocumentType: {
        import_entitlement: {
          version: 2,
          mode: 'positional',
          hasHeader: false,
          columns: {},
          positions: { 'line.gift_level_snapshot': 0 },
          defaults: {},
          columnOrder: ['line.gift_level_snapshot'],
        },
      },
    })
    const mapping = mappingFromPresetForDocumentType(preset, 'import_entitlement')
    expect(mapping).not.toBeNull()
    expect(mapping!.positions?.['line.gift_level_snapshot']).toBe(0)
    expect(mappingFromPresetForDocumentType(preset, 'import_sales_order')).toBeNull()
  })
})
