export type SortOrder = 'ascend' | 'descend' | false

export interface SortDescriptor<T> {
  key: string
  getValue?: (row: T) => string | number | boolean | Date | null | undefined
  compare?: (a: T, b: T) => number
}

function compareValues(a: unknown, b: unknown): number {
  // null/undefined go last
  if (a == null && b == null) return 0
  if (a == null) return 1
  if (b == null) return -1
  // empty string goes last
  if (a === '' && b === '') return 0
  if (a === '') return 1
  if (b === '') return -1
  // dates
  if (a instanceof Date && b instanceof Date) return a.getTime() - b.getTime()
  // numbers
  if (typeof a === 'number' && typeof b === 'number') return a - b
  // booleans: true > false
  if (typeof a === 'boolean' && typeof b === 'boolean') return (a ? 1 : 0) - (b ? 1 : 0)
  // strings with localeCompare
  return String(a).localeCompare(String(b))
}

export function stableSortRows<T>(
  items: T[],
  descriptor: SortDescriptor<T> | null,
  order: SortOrder,
): T[] {
  if (!descriptor || !order) return [...items]

  const result = items.map((item, index) => ({ item, index }))

  result.sort((a, b) => {
    let cmp = 0
    if (descriptor.compare) {
      cmp = descriptor.compare(a.item, b.item)
    } else if (descriptor.getValue) {
      cmp = compareValues(descriptor.getValue(a.item), descriptor.getValue(b.item))
    }
    if (order === 'descend') cmp = -cmp
    // Stable tiebreaker: original index
    if (cmp === 0) cmp = a.index - b.index
    return cmp
  })

  return result.map((r) => r.item)
}
