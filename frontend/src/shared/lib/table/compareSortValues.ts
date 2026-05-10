import { buildKanaRomajiKey, isPureKana, hasHan, hasHangul, toHiragana } from './kanaRomaji.ts'

const defaultCollator = new Intl.Collator(undefined, {
  usage: 'sort',
  numeric: true,
  sensitivity: 'base',
  ignorePunctuation: true,
})

const pinyinCollator = new Intl.Collator('zh-u-co-pinyin', {
  usage: 'sort',
  numeric: true,
  sensitivity: 'base',
  ignorePunctuation: true,
})

const koreanCollator = new Intl.Collator('ko', {
  usage: 'sort',
  numeric: true,
  sensitivity: 'base',
  ignorePunctuation: true,
})

function collatorCompare(a: string, b: string, collator: Intl.Collator): number {
  if (a === '' && b === '') return 0
  if (a === '') return 1
  if (b === '') return -1
  return collator.compare(a, b)
}

export function compareStrings(a: string, b: string): number {
  const hasHangulA = hasHangul(a)
  const hasHangulB = hasHangul(b)
  const hasHanA = hasHan(a)
  const hasHanB = hasHan(b)
  const pureKanaA = isPureKana(a)
  const pureKanaB = isPureKana(b)

  // Hangul-only strings → Korean collator
  if (hasHangulA && !hasHanA && hasHangulB && !hasHanB) {
    return collatorCompare(a, b, koreanCollator)
  }
  if (hasHangulA && !hasHanA) return collatorCompare(a, b, koreanCollator)
  if (hasHangulB && !hasHanB) return collatorCompare(a, b, koreanCollator)

  // Pure kana → romaji comparison
  if (pureKanaA && pureKanaB) {
    const romA = buildKanaRomajiKey(a)
    const romB = buildKanaRomajiKey(b)
    const cmp = defaultCollator.compare(romA, romB)
    if (cmp !== 0) return cmp
    // Tiebreak: hiragana before katakana
    const hiraA = toHiragana(a)
    const hiraB = toHiragana(b)
    if (a !== hiraA && b === hiraB) return 1  // a is katakana, b is hiragana → b first
    if (a === hiraA && b !== hiraB) return -1 // a is hiragana, b is katakana → a first
    return cmp
  }
  if (pureKanaA) return -1
  if (pureKanaB) return 1

  // Han characters → pinyin collator
  if (hasHanA || hasHanB) {
    return collatorCompare(a, b, pinyinCollator)
  }

  // Default
  return collatorCompare(a, b, defaultCollator)
}

export function compareValues(a: unknown, b: unknown): number {
  // null/undefined always sink to bottom
  if (a == null && b == null) return 0
  if (a == null) return 1
  if (b == null) return -1
  // empty string always sinks to bottom
  if (a === '' && b === '') return 0
  if (a === '') return 1
  if (b === '') return -1
  // dates
  if (a instanceof Date && b instanceof Date) return a.getTime() - b.getTime()
  // numbers
  if (typeof a === 'number' && typeof b === 'number') return a - b
  // booleans
  if (typeof a === 'boolean' && typeof b === 'boolean') return (a ? 1 : 0) - (b ? 1 : 0)
  // strings use script-aware comparison
  if (typeof a === 'string' && typeof b === 'string') return compareStrings(a, b)
  // Mixed types or other — safe string conversion
  return compareStrings(String(a), String(b))
}

export function compareForSort(order: 'ascend' | 'descend') {
  return (a: unknown, b: unknown): number => {
    const cmp = compareValues(a, b)
    return order === 'descend' ? -cmp : cmp
  }
}
