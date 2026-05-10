import { assertEquals, assert } from 'jsr:@std/assert'
import { compareValues, compareStrings, classifySortBucket } from './compareSortValues.ts'
import { toHiragana, buildKanaRomajiKey, isPureKana, hasHan, hasHangul } from './kanaRomaji.ts'

// --- kanaRomaji ---
Deno.test('toHiragana converts katakana to hiragana', () => {
  assertEquals(toHiragana('アイウエオ'), 'あいうえお')
  assertEquals(toHiragana('カキクケコ'), 'かきくけこ')
  assertEquals(toHiragana('コンニチハ'), 'こんにちは')
})

Deno.test('toHiragana handles small kana and mixed', () => {
  assertEquals(toHiragana('キャ'), 'きゃ')
  assertEquals(toHiragana('ッ'), 'っ')
})

Deno.test('buildKanaRomajiKey outputs romaji', () => {
  assertEquals(buildKanaRomajiKey('あか'), 'aka')
  assertEquals(buildKanaRomajiKey('ねこ'), 'neko')
})

Deno.test('isPureKana detects pure kana', () => {
  assertEquals(isPureKana('あいう'), true)
  assertEquals(isPureKana('アイウ'), true)
  assertEquals(isPureKana('漢字'), false)
  assertEquals(isPureKana('あ漢'), false)
})

Deno.test('hasHan detects CJK characters', () => {
  assertEquals(hasHan('中文'), true)
  assertEquals(hasHan('日本語'), true)
  assertEquals(hasHan('abc'), false)
})

Deno.test('hasHangul detects Korean', () => {
  assertEquals(hasHangul('한글'), true)
  assertEquals(hasHangul('abc'), false)
})

// --- compareValues ---
Deno.test('compareValues: numbers are sorted naturally', () => {
  // "2" < "19" in natural order
  assertEquals(compareValues(2, 19) < 0, true)
  assertEquals(compareValues(100, 2) > 0, true)
})

Deno.test('compareValues: numeric strings are sorted naturally', () => {
  assertEquals(compareValues('2', '19') < 0, true)
  assertEquals(compareValues('A2', 'A19') < 0, true)
})

Deno.test('compareValues: null/undefined sink to bottom', () => {
  assertEquals(compareValues(null, 'abc') > 0, true)
  assertEquals(compareValues('abc', null) < 0, true)
  assertEquals(compareValues(undefined, 'abc') > 0, true)
})

Deno.test('compareValues: empty string sinks to bottom', () => {
  assertEquals(compareValues('', 'abc') > 0, true)
  assertEquals(compareValues('abc', '') < 0, true)
})

Deno.test('compareValues: Chinese sorted by pinyin', () => {
  // 张 should come after 李 in pinyin (zhang > li)
  assertEquals(compareValues('李', '张') < 0, true)
})

Deno.test('compareValues: pure kana sorted by romaji', () => {
  assertEquals(compareValues('あか', 'いき') < 0, true) // aka < iki
})

Deno.test('compareValues: hiragana before katakana as tiebreak', () => {
  assertEquals(compareValues('あか', 'アカ') < 0, true)
})

Deno.test('compareValues: Korean sorted correctly', () => {
  assertEquals(compareValues('가', '나') < 0, true)
})

Deno.test('compareValues: booleans', () => {
  assertEquals(compareValues(false, true) < 0, true)
})

// --- stableSortRows integration ---
import { stableSortRows } from './stableSortRows.ts'

Deno.test('stableSortRows: ascending puts values in order, nulls at end', () => {
  const rows = [{ name: 'b' }, { name: null }, { name: 'a' }, { name: '' }]
  const result = stableSortRows(rows, { key: 'name', getValue: (r: any) => r.name }, 'ascend')
  assertEquals(result[0].name, 'a')
  assertEquals(result[1].name, 'b')
  assertEquals(result[2].name, null)
  assertEquals(result[3].name, '')
})

Deno.test('stableSortRows: descending puts values in reverse, nulls still at end', () => {
  const rows = [{ name: 'b' }, { name: null }, { name: 'a' }, { name: '' }]
  const result = stableSortRows(rows, { key: 'name', getValue: (r: any) => r.name }, 'descend')
  assertEquals(result[0].name, 'b')
  assertEquals(result[1].name, 'a')
  assertEquals(result[2].name, null)
  assertEquals(result[3].name, '')
})

Deno.test('stableSortRows: preserves original order on equal values', () => {
  const rows = [{ id: 1, v: 'same' }, { id: 2, v: 'same' }, { id: 3, v: 'same' }]
  const result = stableSortRows(rows, { key: 'v', getValue: (r: any) => r.v }, 'ascend')
  assertEquals(result[0].id, 1)
  assertEquals(result[1].id, 2)
  assertEquals(result[2].id, 3)
})

Deno.test('stableSortRows: descending preserves original order on equal values', () => {
  const rows = [{ id: 1, v: 'same' }, { id: 2, v: 'same' }, { id: 3, v: 'same' }]
  const result = stableSortRows(rows, { key: 'v', getValue: (r: any) => r.v }, 'descend')
  assertEquals(result[0].id, 1)
  assertEquals(result[1].id, 2)
  assertEquals(result[2].id, 3)
})

// --- bucket classification ---
Deno.test('classifySortBucket: digit', () => {
  assertEquals(classifySortBucket('123'), 'digit')
  assertEquals(classifySortBucket('０１２'), 'digit')
})

Deno.test('classifySortBucket: latin', () => {
  assertEquals(classifySortBucket('ABC'), 'latin')
  assertEquals(classifySortBucket('hello'), 'latin')
})

Deno.test('classifySortBucket: han', () => {
  assertEquals(classifySortBucket('张三'), 'han')
  assertEquals(classifySortBucket('日本語'), 'han') // kanji → han
})

Deno.test('classifySortBucket: hiragana', () => {
  assertEquals(classifySortBucket('あいう'), 'hiragana')
})

Deno.test('classifySortBucket: katakana', () => {
  assertEquals(classifySortBucket('アイウ'), 'katakana')
})

Deno.test('classifySortBucket: hangul', () => {
  assertEquals(classifySortBucket('한글'), 'hangul')
})

Deno.test('classifySortBucket: leading bracket skipped', () => {
  assertEquals(classifySortBucket('【张三】'), 'han')
})

Deno.test('classifySortBucket: A12中文 → latin (first strong char is A)', () => {
  assertEquals(classifySortBucket('A12中文'), 'latin')
})

Deno.test('classifySortBucket: 123abc → digit', () => {
  assertEquals(classifySortBucket('123abc'), 'digit')
})

// --- bucket ordering ---
Deno.test('compareStrings: bucket order ascending (digit < latin < han < hiragana < katakana < hangul < other)', () => {
  assert(compareStrings('1', 'A') < 0)
  assert(compareStrings('A', '张') < 0)
  assert(compareStrings('张', 'あ') < 0)
  assert(compareStrings('あ', 'ア') < 0)
  assert(compareStrings('ア', '가') < 0)
  assert(compareStrings('가', '#') < 0)
})

Deno.test('compareStrings: bucket order is preserved regardless of numeric value', () => {
  // 1000 (digit) should come before A (latin)
  assert(compareStrings('1000', 'A') < 0)
})

// --- mixed natural sort ---
Deno.test('compareStrings: numeric strings sort naturally', () => {
  assert(compareStrings('2', '19') < 0)
  assert(compareStrings('A2', 'A19') < 0)
  assert(compareStrings('SKU-2', 'SKU-19') < 0)
})

Deno.test('compareStrings: mixed CJK with numbers sort naturally', () => {
  assert(compareStrings('张2', '张19') < 0)
  assert(compareStrings('第2名', '第19名') < 0)
})

Deno.test('compareStrings: mixed kana with numbers sort naturally', () => {
  assert(compareStrings('あ2', 'あ19') < 0)
})

Deno.test('compareStrings: mixed hangul with numbers sort naturally', () => {
  assert(compareStrings('가2', '가19') < 0)
})

Deno.test('stableSortRows: mixed natural sort descending', () => {
  const rows = [{ v: 'A19' }, { v: 'A2' }, { v: 'A1' }]
  const result = stableSortRows(rows, { key: 'v', getValue: (r: any) => r.v }, 'descend')
  assertEquals(result[0].v, 'A19')
  assertEquals(result[1].v, 'A2')
  assertEquals(result[2].v, 'A1')
})

Deno.test('stableSortRows: mixed natural sort keeps nulls at end in descend', () => {
  const rows = [{ v: 'A2' }, { v: null }, { v: 'A19' }]
  const result = stableSortRows(rows, { key: 'v', getValue: (r: any) => r.v }, 'descend')
  assertEquals(result[0].v, 'A19')
  assertEquals(result[1].v, 'A2')
  assertEquals(result[2].v, null)
})
