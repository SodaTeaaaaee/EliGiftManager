import { describe, expect, test } from 'vitest'
import type { TemplateConfig } from '@/entities/models'
import {
  SEMANTIC_GROUPS,
  buildSampleEditorRows,
  defaultVisibleKeysFor,
  inEffectTemplateIDs,
  semanticGroupOf,
} from './templateCatalog'

const DICTIONARY = [
  'customer.display_name',
  'identity.platform',
  'identity.value',
  'identity.type',
  'membership.level',
  'source.document_no',
  'source.line_no',
  'source.created_at',
  'product.alias_id',
  'product.alias_title',
  'product.alias_spec',
  'product.factory_sku',
  'product.name',
  'quantity',
  'recipient.name',
  'recipient.phone',
  'recipient.country',
  'recipient.province',
  'recipient.city',
  'recipient.district',
  'recipient.address_line1',
  'recipient.address_line2',
  'recipient.postal_code',
  'tracking.id',
  'shipment.tracking_no',
  'shipment.carrier_code',
  'shipment.carrier_name',
  'shipment.shipped_at',
  'shipment.quantity',
]

describe('semantic dictionary groups', () => {
  test('every dictionary key lands in one of the five groups by prefix', () => {
    for (const key of DICTIONARY) {
      expect(SEMANTIC_GROUPS).toContain(semanticGroupOf(key))
    }
    expect(semanticGroupOf('customer.display_name')).toBe('identity')
    expect(semanticGroupOf('membership.level')).toBe('identity')
    expect(semanticGroupOf('source.line_no')).toBe('source')
    expect(semanticGroupOf('product.factory_sku')).toBe('product')
    expect(semanticGroupOf('recipient.postal_code')).toBe('recipient')
    expect(semanticGroupOf('tracking.id')).toBe('execution')
    expect(semanticGroupOf('shipment.quantity')).toBe('execution')
  })

  test('bare quantity belongs to 商品与数量', () => {
    expect(semanticGroupOf('quantity')).toBe('product')
  })
})

describe('default visible field set per document type', () => {
  test('membership_list shows identity, level and display name only', () => {
    expect(defaultVisibleKeysFor('membership_list', DICTIONARY)).toEqual([
      'customer.display_name',
      'identity.platform',
      'identity.value',
      'identity.type',
      'membership.level',
    ])
  })

  test('order_export shows source, identity value, aliases, quantity and recipient', () => {
    const keys = defaultVisibleKeysFor('order_export', DICTIONARY) ?? []
    expect(keys).toContain('source.document_no')
    expect(keys).toContain('identity.value')
    expect(keys).toContain('product.alias_spec')
    expect(keys).toContain('quantity')
    expect(keys).toContain('recipient.address_line1')
    expect(keys).not.toContain('product.factory_sku')
    expect(keys).not.toContain('membership.level')
    expect(keys).not.toContain('shipment.tracking_no')
  })

  test('shipment_return shows tracking, shipment, aliases, recipient and source time', () => {
    const keys = defaultVisibleKeysFor('shipment_return', DICTIONARY) ?? []
    expect(keys).toContain('tracking.id')
    expect(keys).toContain('shipment.carrier_name')
    expect(keys).toContain('product.alias_title')
    expect(keys).toContain('recipient.phone')
    expect(keys).toContain('source.created_at')
    expect(keys).not.toContain('source.document_no')
    expect(keys).not.toContain('identity.value')
  })

  test('unknown document types show everything', () => {
    expect(defaultVisibleKeysFor('factory_order', DICTIONARY)).toBeNull()
  })
})

function tpl(partial: Partial<TemplateConfig>): TemplateConfig {
  return {
    ID: 0,
    PlatformID: 1,
    DocumentType: 'factory_order',
    Direction: 'output',
    Name: 't',
    Version: 1,
    Builtin: false,
    ...partial,
  }
}

describe('currently in effect rule', () => {
  const auto = new Set(['factory_order', 'writeback'])

  test('greatest UpdatedAt wins within one platform/direction/type slot', () => {
    const ids = inEffectTemplateIDs(
      [
        tpl({ ID: 1, UpdatedAt: '2026-01-01T00:00:00Z' }),
        tpl({ ID: 2, UpdatedAt: '2026-03-01T00:00:00Z' }),
        tpl({ ID: 3, UpdatedAt: '2026-02-01T00:00:00Z' }),
      ],
      auto,
    )
    expect([...ids]).toEqual([2])
  })

  test('ties on UpdatedAt break on the greatest ID', () => {
    const ids = inEffectTemplateIDs(
      [
        tpl({ ID: 7, UpdatedAt: '2026-03-01T00:00:00Z' }),
        tpl({ ID: 5, UpdatedAt: '2026-03-01T00:00:00Z' }),
      ],
      auto,
    )
    expect([...ids]).toEqual([7])
  })

  test('slots are independent per platform and document type; builtins never count', () => {
    const ids = inEffectTemplateIDs(
      [
        tpl({ ID: 1, PlatformID: 1, UpdatedAt: '2026-01-01T00:00:00Z' }),
        tpl({ ID: 2, PlatformID: 2, UpdatedAt: '2026-01-01T00:00:00Z' }),
        tpl({ ID: 3, PlatformID: 1, DocumentType: 'writeback', UpdatedAt: '2026-01-01T00:00:00Z' }),
        tpl({ ID: 4, PlatformID: 1, Builtin: true, UpdatedAt: '2027-01-01T00:00:00Z' }),
      ],
      auto,
    )
    expect([...ids].sort()).toEqual([1, 2, 3])
  })

  test('user-picked document types never get the badge', () => {
    const ids = inEffectTemplateIDs(
      [tpl({ ID: 1, DocumentType: 'shipment_return', Direction: 'input', UpdatedAt: '2026-01-01T00:00:00Z' })],
      auto,
    )
    expect(ids.size).toBe(0)
  })
})

describe('sample file → editor rows', () => {
  const records = [
    ['等级', 'uid', '昵称'],
    ['舰长', '1001', 'Aki'],
    ['提督', '1002'],
  ]

  test('header mode keys rows by the first record and skips it as data', () => {
    const { sourceHeaders, sampleRows } = buildSampleEditorRows({ Records: records }, true)
    expect(sourceHeaders).toEqual(['等级', 'uid', '昵称'])
    expect(sampleRows).toEqual([
      { 等级: '舰长', uid: '1001', 昵称: 'Aki' },
      { 等级: '提督', uid: '1002', 昵称: '' },
    ])
  })

  test('positional mode keys every record by 0-based index strings', () => {
    const { sourceHeaders, sampleRows } = buildSampleEditorRows({ Records: records }, false)
    expect(sourceHeaders).toEqual(['0', '1', '2'])
    expect(sampleRows).toHaveLength(3)
    expect(sampleRows[0]).toEqual({ '0': '等级', '1': 'uid', '2': '昵称' })
    expect(sampleRows[2]).toEqual({ '0': '提督', '1': '1002', '2': '' })
  })

  test('blank header cells fall back to their column index', () => {
    const { sourceHeaders } = buildSampleEditorRows({ Records: [['a', ' ', 'c'], ['1', '2', '3']] }, true)
    expect(sourceHeaders).toEqual(['a', '1', 'c'])
  })

  test('empty samples yield empty inputs', () => {
    expect(buildSampleEditorRows(null, true)).toEqual({ sourceHeaders: [], sampleRows: [] })
    expect(buildSampleEditorRows({ Records: [] }, false)).toEqual({ sourceHeaders: [], sampleRows: [] })
  })
})
