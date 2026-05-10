import { compareValues } from './compareSortValues'

export type SortOrder = 'ascend' | 'descend' | false

export interface SortDescriptor<T> {
  key: string
  getValue?: (row: T) => string | number | boolean | Date | null | undefined
  compare?: (a: T, b: T) => number
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
