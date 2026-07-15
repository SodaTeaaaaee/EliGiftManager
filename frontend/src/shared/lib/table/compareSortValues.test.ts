import { assert, expect, test } from 'vitest'
import {
  classifySortBucket,
  compareStrings,
  compareValues,
} from './compareSortValues.ts'
import {
  buildKanaRomajiKey,
  hasHan,
  hasHangul,
  isPureKana,
  toHiragana,
} from './kanaRomaji.ts'
import { stableSortRows } from './stableSortRows.ts'

// --- kanaRomaji ---
test('toHiragana converts katakana to hiragana', () => {
  expect(toHiragana('アイウエオ')).toBe('あいうえお')
  expect(toHiragana('カキクケコ')).toBe('かきくけこ')
  expect(toHiragana('コンニチハ')).toBe('こんにちは')
})

test('toHiragana handles small kana and mixed', () => {
  expect(toHiragana('キャ')).toBe('きゃ')
  expect(toHiragana('ッ')).toBe('っ')
})

test('buildKanaRomajiKey outputs romaji', () => {
  expect(buildKanaRomajiKey('あか')).toBe('aka')
  expect(buildKanaRomajiKey('ねこ')).toBe('neko')
})

test('isPureKana detects pure kana', () => {
  expect(isPureKana('あいう')).toBe(true)
  expect(isPureKana('アイウ')).toBe(true)
  expect(isPureKana('漢字')).toBe(false)
  expect(isPureKana('あ漢')).toBe(false)
})

test('hasHan detects CJK characters', () => {
  expect(hasHan('中文')).toBe(true)
  expect(hasHan('日本語')).toBe(true)
  expect(hasHan('abc')).toBe(false)
})

test('hasHangul detects Korean', () => {
  expect(hasHangul('한글')).toBe(true)
  expect(hasHangul('abc')).toBe(false)
})

// --- compareValues ---
test('compareValues: numbers are sorted naturally', () => {
  // "2" < "19" in natural order
  expect(compareValues(2, 19) < 0).toBe(true)
  expect(compareValues(100, 2) > 0).toBe(true)
})

test('compareValues: numeric strings are sorted naturally', () => {
  expect(compareValues('2', '19') < 0).toBe(true)
  expect(compareValues('A2', 'A19') < 0).toBe(true)
})

test('compareValues: null/undefined sink to bottom', () => {
  expect(compareValues(null, 'abc') > 0).toBe(true)
  expect(compareValues('abc', null) < 0).toBe(true)
  expect(compareValues(undefined, 'abc') > 0).toBe(true)
})

test('compareValues: empty string sinks to bottom', () => {
  expect(compareValues('', 'abc') > 0).toBe(true)
  expect(compareValues('abc', '') < 0).toBe(true)
})

test('compareValues: Chinese sorted by pinyin', () => {
  // 张 should come after 李 in pinyin (zhang > li)
  expect(compareValues('李', '张') < 0).toBe(true)
})

test('compareValues: pure kana sorted by romaji', () => {
  expect(compareValues('あか', 'いき') < 0).toBe(true) // aka < iki
})

test('compareValues: hiragana before katakana as tiebreak', () => {
  expect(compareValues('あか', 'アカ') < 0).toBe(true)
})

test('compareValues: Korean sorted correctly', () => {
  expect(compareValues('가', '나') < 0).toBe(true)
})

test('compareValues: booleans', () => {
  expect(compareValues(false, true) < 0).toBe(true)
})

// --- stableSortRows integration ---
test('stableSortRows: ascending puts values in order, nulls at end', () => {
  const rows = [{ name: 'b' }, { name: null }, { name: 'a' }, { name: '' }]
  const result = stableSortRows(rows, {
    key: 'name',
    getValue: (r: { name: string | null }) => r.name,
  }, 'ascend')
  expect(result[0].name).toBe('a')
  expect(result[1].name).toBe('b')
  expect(result[2].name).toBe(null)
  expect(result[3].name).toBe('')
})

test('stableSortRows: descending puts values in reverse, nulls still at end', () => {
  const rows = [{ name: 'b' }, { name: null }, { name: 'a' }, { name: '' }]
  const result = stableSortRows(rows, {
    key: 'name',
    getValue: (r: { name: string | null }) => r.name,
  }, 'descend')
  expect(result[0].name).toBe('b')
  expect(result[1].name).toBe('a')
  expect(result[2].name).toBe(null)
  expect(result[3].name).toBe('')
})

test('stableSortRows: preserves original order on equal values', () => {
  const rows = [{ id: 1, v: 'same' }, { id: 2, v: 'same' }, {
    id: 3,
    v: 'same',
  }]
  const result = stableSortRows(
    rows,
    { key: 'v', getValue: (r: { id: number; v: string }) => r.v },
    'ascend',
  )
  expect(result[0].id).toBe(1)
  expect(result[1].id).toBe(2)
  expect(result[2].id).toBe(3)
})

test('stableSortRows: descending preserves original order on equal values', () => {
  const rows = [{ id: 1, v: 'same' }, { id: 2, v: 'same' }, {
    id: 3,
    v: 'same',
  }]
  const result = stableSortRows(
    rows,
    { key: 'v', getValue: (r: { id: number; v: string }) => r.v },
    'descend',
  )
  expect(result[0].id).toBe(1)
  expect(result[1].id).toBe(2)
  expect(result[2].id).toBe(3)
})

// --- bucket classification ---
test('classifySortBucket: digit', () => {
  expect(classifySortBucket('123')).toBe('digit')
  expect(classifySortBucket('０１２')).toBe('digit')
})

test('classifySortBucket: latin', () => {
  expect(classifySortBucket('ABC')).toBe('latin')
  expect(classifySortBucket('hello')).toBe('latin')
})

test('classifySortBucket: han', () => {
  expect(classifySortBucket('张三')).toBe('han')
  expect(classifySortBucket('日本語')).toBe('han') // kanji → han
})

test('classifySortBucket: hiragana', () => {
  expect(classifySortBucket('あいう')).toBe('hiragana')
})

test('classifySortBucket: katakana', () => {
  expect(classifySortBucket('アイウ')).toBe('katakana')
})

test('classifySortBucket: hangul', () => {
  expect(classifySortBucket('한글')).toBe('hangul')
})

test('classifySortBucket: leading bracket skipped', () => {
  expect(classifySortBucket('【张三】')).toBe('han')
})

test('classifySortBucket: A12中文 → latin (first strong char is A)', () => {
  expect(classifySortBucket('A12中文')).toBe('latin')
})

test('classifySortBucket: 123abc → digit', () => {
  expect(classifySortBucket('123abc')).toBe('digit')
})

// --- bucket ordering ---
test('compareStrings: bucket order ascending (digit < latin < han < hiragana < katakana < hangul < other)', () => {
  assert(compareStrings('1', 'A') < 0)
  assert(compareStrings('A', '张') < 0)
  assert(compareStrings('张', 'あ') < 0)
  assert(compareStrings('あ', 'ア') < 0)
  assert(compareStrings('ア', '가') < 0)
  assert(compareStrings('가', '#') < 0)
})

test('compareStrings: bucket order is preserved regardless of numeric value', () => {
  // 1000 (digit) should come before A (latin)
  assert(compareStrings('1000', 'A') < 0)
})

// --- mixed natural sort ---
test('compareStrings: numeric strings sort naturally', () => {
  assert(compareStrings('2', '19') < 0)
  assert(compareStrings('A2', 'A19') < 0)
  assert(compareStrings('SKU-2', 'SKU-19') < 0)
})

test('compareStrings: mixed CJK with numbers sort naturally', () => {
  assert(compareStrings('张2', '张19') < 0)
  assert(compareStrings('第2名', '第19名') < 0)
})

test('compareStrings: mixed kana with numbers sort naturally', () => {
  assert(compareStrings('あ2', 'あ19') < 0)
})

test('compareStrings: mixed hangul with numbers sort naturally', () => {
  assert(compareStrings('가2', '가19') < 0)
})

test('stableSortRows: mixed natural sort descending', () => {
  const rows = [{ v: 'A19' }, { v: 'A2' }, { v: 'A1' }]
  const result = stableSortRows(
    rows,
    { key: 'v', getValue: (r: { v: string }) => r.v },
    'descend',
  )
  expect(result[0].v).toBe('A19')
  expect(result[1].v).toBe('A2')
  expect(result[2].v).toBe('A1')
})

test('stableSortRows: mixed natural sort keeps nulls at end in descend', () => {
  const rows = [{ v: 'A2' }, { v: null }, { v: 'A19' }]
  const result = stableSortRows(
    rows,
    { key: 'v', getValue: (r: { v: string | null }) => r.v },
    'descend',
  )
  expect(result[0].v).toBe('A19')
  expect(result[1].v).toBe('A2')
  expect(result[2].v).toBe(null)
})
