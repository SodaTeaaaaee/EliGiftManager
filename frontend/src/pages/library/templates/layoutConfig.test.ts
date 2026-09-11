import { describe, expect, test } from 'vitest'
import {
  emptyLayoutConfig,
  moveLayoutColumn,
  parseLayoutConfig,
  placeholderSampleFor,
  serializeLayoutConfig,
} from './layoutConfig'

describe('layout config contract', () => {
  test('serializes to the backend LayoutConfig v1 shape', () => {
    const raw = serializeLayoutConfig({
      version: 1,
      format: 'xlsx',
      columns: [
        { key: 'tracking.id', header: '第三方订单号' },
        { key: 'recipient.name', header: '  ' },
        { key: 'quantity', header: '下单数量' },
      ],
    })
    expect(JSON.parse(raw)).toEqual({
      version: 1,
      format: 'xlsx',
      columnOrder: ['tracking.id', 'recipient.name', 'quantity'],
      headerNames: { 'tracking.id': '第三方订单号', quantity: '下单数量' },
    })
  })

  test('omits headerNames entirely when every header is blank', () => {
    const stored = JSON.parse(
      serializeLayoutConfig({ version: 1, format: 'csv', columns: [{ key: 'quantity', header: '' }] }),
    ) as Record<string, unknown>
    expect('headerNames' in stored).toBe(false)
    expect(stored.columnOrder).toEqual(['quantity'])
  })

  test('drops blank and duplicate keys on serialize', () => {
    const stored = JSON.parse(
      serializeLayoutConfig({
        version: 1,
        format: 'csv',
        columns: [
          { key: 'quantity', header: '' },
          { key: ' ', header: 'x' },
          { key: 'quantity', header: 'dup' },
        ],
      }),
    ) as Record<string, unknown>
    expect(stored.columnOrder).toEqual(['quantity'])
  })

  test('parses stored LayoutJSON back into ordered columns with headers', () => {
    const parsed = parseLayoutConfig(
      JSON.stringify({
        version: 1,
        format: 'csv',
        columnOrder: ['tracking.id', 'recipient.name', 'tracking.id'],
        headerNames: { 'tracking.id': '第三方订单号' },
      }),
    )
    expect(parsed.format).toBe('csv')
    expect(parsed.columns).toEqual([
      { key: 'tracking.id', header: '第三方订单号' },
      { key: 'recipient.name', header: '' },
    ])
  })

  test('round-trips through serialize and parse', () => {
    const original = {
      version: 1,
      format: 'xlsx' as const,
      columns: [
        { key: 'shipment.tracking_no', header: '物流单号' },
        { key: 'shipment.carrier_code', header: '' },
      ],
    }
    expect(parseLayoutConfig(serializeLayoutConfig(original))).toEqual(original)
  })

  test('empty, invalid, or foreign-version input yields an empty v1 layout', () => {
    expect(parseLayoutConfig('')).toEqual(emptyLayoutConfig())
    expect(parseLayoutConfig(null)).toEqual(emptyLayoutConfig())
    expect(parseLayoutConfig('not json')).toEqual(emptyLayoutConfig())
    expect(parseLayoutConfig(JSON.stringify({ version: 2, format: 'csv', columnOrder: ['a'] }))).toEqual(
      emptyLayoutConfig(),
    )
  })

  test('unknown formats normalize to csv', () => {
    const parsed = parseLayoutConfig(JSON.stringify({ version: 1, format: 'pdf', columnOrder: [] }))
    expect(parsed.format).toBe('csv')
  })

  test('moveLayoutColumn swaps neighbours and ignores edges', () => {
    const columns = [
      { key: 'a', header: '' },
      { key: 'b', header: '' },
      { key: 'c', header: '' },
    ]
    expect(moveLayoutColumn(columns, 1, -1).map((c) => c.key)).toEqual(['b', 'a', 'c'])
    expect(moveLayoutColumn(columns, 2, 1)).toBe(columns)
    expect(moveLayoutColumn(columns, 0, -1)).toBe(columns)
    // The input list is never mutated.
    expect(columns.map((c) => c.key)).toEqual(['a', 'b', 'c'])
  })

  test('placeholder samples prefer realistic identifiers and fall back to the label', () => {
    expect(placeholderSampleFor('quantity', '数量')).toBe('1')
    expect(placeholderSampleFor('shipment.carrier_code', '承运商代号')).toBe('SF')
    expect(placeholderSampleFor('recipient.name', '收件人姓名')).toBe('〈收件人姓名〉')
  })
})
