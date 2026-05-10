import { buildKanaRomajiKey, toHiragana } from './kanaRomaji.ts'

type SortBucket = 'digit' | 'latin' | 'han' | 'hiragana' | 'katakana' | 'hangul' | 'other'

const BUCKET_ORDER_ASCEND: Record<SortBucket, number> = {
  digit: 0,
  latin: 1,
  han: 2,
  hiragana: 3,
  katakana: 4,
  hangul: 5,
  other: 6,
}

function getUnicodeCategory(cp: number): string {
  const ch = String.fromCodePoint(cp)
  if (/\p{Z}/u.test(ch)) {
    if (cp === 0x20 || cp === 0xA0 || (cp >= 0x2000 && cp <= 0x200A)) return 'Zs'
    if (cp === 0x2028) return 'Zl'
    if (cp === 0x2029) return 'Zp'
    return 'Zs'
  }
  if (/\p{P}/u.test(ch)) {
    if (/[\p{Ps}\p{Pi}]/u.test(ch)) return 'Ps'
    if (/[\p{Pe}\p{Pf}]/u.test(ch)) return 'Pe'
    return 'Po'
  }
  if (/\p{M}/u.test(ch)) {
    if (/\p{Mn}/u.test(ch)) return 'Mn'
    if (/\p{Mc}/u.test(ch)) return 'Mc'
    return 'Me'
  }
  if (/\p{Sk}/u.test(ch)) return 'Sk'
  if (/\p{Lm}/u.test(ch)) return 'Lm'
  if (cp < 0x20 || (cp >= 0x7F && cp <= 0x9F)) return 'Cc'
  return 'Other'
}

function getScriptBucket(str: string): SortBucket {
  if (str.length === 0) return 'other'
  for (let i = 0; i < str.length; i++) {
    const ch = str[i]
    const cp = ch.codePointAt(0)!
    if (cp <= 0x20 || (cp >= 0x2000 && cp <= 0x200D) || cp === 0xFEFF) continue
    if (cp === 0x300C || cp === 0x300D || cp === 0x3010 || cp === 0x3011 || cp === 0xFF3B || cp === 0xFF3D) continue
    if (cp === 0x300E || cp === 0x300F || cp === 0x300A || cp === 0x300B) continue
    if (cp >= 0xFF01 && cp <= 0xFF0F) continue
    if (cp >= 0xFF1A && cp <= 0xFF20) continue
    if (cp >= 0xFF3B && cp <= 0xFF40) continue
    const cat = getUnicodeCategory(cp)
    if (cat === 'Mn' || cat === 'Mc' || cat === 'Me' || cat === 'Sk' || cat === 'Lm') continue
    if (/\p{Nd}/u.test(ch)) return 'digit'
    if (cp >= 0x30 && cp <= 0x39) return 'digit'
    if (cp >= 0xFF10 && cp <= 0xFF19) return 'digit'
    if (/\p{Script=Latin}/u.test(ch)) return 'latin'
    if (/\p{Script=Han}/u.test(ch)) return 'han'
    if (cp >= 0x3040 && cp <= 0x309F) return 'hiragana'
    if (cp >= 0x30A0 && cp <= 0x30FF) return 'katakana'
    if (/\p{Script=Hangul}/u.test(ch)) return 'hangul'
    return 'other'
  }
  return 'other'
}

export function classifySortBucket(str: string | null | undefined): SortBucket | null {
  if (str == null || str === '') return null
  return getScriptBucket(str)
}

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
  const bucketA = classifySortBucket(a)
  const bucketB = classifySortBucket(b)
  // Different buckets → compare by bucket order
  if (bucketA && bucketB && bucketA !== bucketB) {
    return BUCKET_ORDER_ASCEND[bucketA] - BUCKET_ORDER_ASCEND[bucketB]
  }
  if (bucketA && !bucketB) return -1
  if (!bucketA && bucketB) return 1
  if (!bucketA && !bucketB) return 0

  // Same bucket → intra-bucket comparison
  const bucket = bucketA!
  switch (bucket) {
    case 'digit':
      return defaultCollator.compare(a, b)
    case 'latin':
      return defaultCollator.compare(a, b)
    case 'han':
      return collatorCompare(a, b, pinyinCollator)
    case 'hiragana':
    case 'katakana': {
      const romA = buildKanaRomajiKey(a)
      const romB = buildKanaRomajiKey(b)
      const cmp = defaultCollator.compare(romA, romB)
      if (cmp !== 0) return cmp
      // Within same bucket, but if mixed hira/katakana text, hiragana first
      const hiraA = toHiragana(a)
      const hiraB = toHiragana(b)
      if (a !== hiraA && b === hiraB) return 1
      if (a === hiraA && b !== hiraB) return -1
      return cmp
    }
    case 'hangul':
      return collatorCompare(a, b, koreanCollator)
    default:
      return collatorCompare(a, b, defaultCollator)
  }
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
    const aMissing = a == null || a === ''
    const bMissing = b == null || b === ''
    if (aMissing && bMissing) return 0
    if (aMissing) return 1
    if (bMissing) return -1
    const cmp = compareValues(a, b)
    return order === 'descend' ? -cmp : cmp
  }
}
