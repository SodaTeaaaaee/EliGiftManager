import { describe, expect, test } from 'vitest'
import { applyMapping, applyPreviewTransforms } from './previewTransform'
import { parseMappingRules, serializeMappingRules, type FieldMappingValue } from './types'

describe('mapping preview contract', () => {
  test('applies local transforms in order', () => {
    expect(applyPreviewTransforms("  '00123'  ", ['trim', 'strip_quotes'])).toBe('00123')
    expect(applyPreviewTransforms('"a b"', ['strip_quotes', 'trim'])).toBe('a b')
  })

  test('strip_quotes mirrors the backend: leading apostrophe and CR/LF noise', () => {
    // Spreadsheet text-forcing apostrophe without a closing quote.
    expect(applyPreviewTransforms("'435167587794147", ['strip_quotes'])).toBe('435167587794147')
    // Excel-style CR/LF noise inside identifier cells is dropped everywhere.
    expect(applyPreviewTransforms('4351\r\n6758\r7794', ['strip_quotes'])).toBe('435167587794')
    // Paired quotes strip first; the remaining leading apostrophe still peels.
    expect(applyPreviewTransforms("''4351'", ['strip_quotes'])).toBe('4351')
    expect(applyPreviewTransforms('"4351\r\n"', ['strip_quotes'])).toBe('4351')
  })

  test('context-dependent transformers pass through the local preview', () => {
    expect(applyPreviewTransforms('2024/1/5', ['parseDate'])).toBe('2024/1/5')
    expect(applyPreviewTransforms('普通会员', ['mapEnum'])).toBe('普通会员')
  })

  test('joinAddress merges unit-separated parts and drops empties', () => {
    expect(applyPreviewTransforms(' 北京市 \x1f\x1f海淀区 \x1f ', ['joinAddress'])).toBe('北京市海淀区')
  })

  test('preview mirrors the backend resolution order: source → join → default → transform', () => {
    const mapping: FieldMappingValue = {
      version: 3,
      mode: 'header',
      hasHeader: true,
      columns: { 'shipment.tracking_no': '物流单号', 'recipient.city': '城市' },
      defaults: { 'shipment.carrier_code': ' SF ' },
      transforms: {
        'shipment.tracking_no': ['strip_quotes'],
        'shipment.carrier_code': ['trim'],
      },
      joinSources: { 'recipient.address_line1': ['省', '城市'] },
    }
    const [row] = applyMapping(
      [{ 物流单号: "'123456789012345678'", 城市: '海淀区', 省: '北京市' }],
      mapping,
      ['物流单号', '城市', '省'],
    )
    expect(row.values['shipment.tracking_no']).toBe('123456789012345678')
    expect(row.values['shipment.carrier_code']).toBe('SF')
    expect(row.values['recipient.address_line1']).toBe('北京市海淀区')
    expect(row.values['recipient.city']).toBe('海淀区')
  })

  test('positional mode reads cells by index over sourceHeaders order', () => {
    const mapping: FieldMappingValue = {
      version: 3,
      mode: 'positional',
      hasHeader: false,
      columns: {},
      positions: { quantity: 1 },
      defaults: {},
    }
    const [row] = applyMapping([{ a: 'x', b: '3' }], mapping, ['a', 'b'])
    expect(row.values.quantity).toBe('3')
  })

  test('required, fingerprint, joinSources, and enumMaps survive v3 round-trip', () => {
    const raw = serializeMappingRules({
      version: 3,
      mode: 'header',
      hasHeader: true,
      columns: { 'tracking.id': '追踪号*' },
      defaults: { quantity: '1' },
      transforms: { 'tracking.id': ['trim'] },
      enumMaps: { 'membership.level': { 舰长: 'captain' } },
      joinSources: { 'recipient.address_line1': ['省', '市'] },
      splitSkuQuantity: 'product.alias_id',
      required: ['tracking.id'],
      fingerprint: ['tracking.id', 'identity.value'],
      sheetName: '发货信息',
    })
    const parsed = parseMappingRules(raw)
    expect(parsed.version).toBe(3)
    expect(parsed.transforms?.['tracking.id']).toEqual(['trim'])
    expect(parsed.required).toEqual(['tracking.id'])
    expect(parsed.fingerprint).toEqual(['tracking.id', 'identity.value'])
    expect(parsed.enumMaps?.['membership.level']).toEqual({ 舰长: 'captain' })
    expect(parsed.joinSources?.['recipient.address_line1']).toEqual(['省', '市'])
    expect(parsed.splitSkuQuantity).toBe('product.alias_id')
    expect(parsed.sheetName).toBe('发货信息')
  })

  test('empty optional maps are omitted and incomplete enum rows dropped', () => {
    const raw = serializeMappingRules({
      version: 3,
      mode: 'header',
      hasHeader: true,
      columns: { quantity: '数量' },
      defaults: {},
      enumMaps: { 'membership.level': { 舰长: 'captain', '': 'dropped', 空: '' } },
    })
    const stored = JSON.parse(raw) as Record<string, unknown>
    expect('defaults' in stored).toBe(false)
    expect('positions' in stored).toBe(false)
    expect(stored.enumMaps).toEqual({ 'membership.level': { 舰长: 'captain' } })
  })

  test('invalid JSON or foreign schema versions yield an empty v3 mapping', () => {
    expect(parseMappingRules('not json').version).toBe(3)
    expect(parseMappingRules('not json').columns).toEqual({})
    const v2 = parseMappingRules(JSON.stringify({ version: 2, mode: 'header', columns: { a: 'b' } }))
    expect(v2.columns).toEqual({})
    expect(v2.version).toBe(3)
    expect(parseMappingRules('')).toEqual(parseMappingRules(null))
  })
})
