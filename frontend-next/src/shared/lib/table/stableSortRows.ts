import { compareValues } from "./compareSortValues.ts";

export type SortOrder = "ascend" | "descend" | false;

export interface SortDescriptor<T> {
  key: string;
  getValue?: (row: T) => string | number | boolean | Date | null | undefined;
  compare?: (a: T, b: T) => number;
}

export function stableSortRows<T>(
  items: T[],
  descriptor: SortDescriptor<T> | null,
  order: SortOrder,
): T[] {
  if (!descriptor || !order) return [...items];

  const result = items.map((item, index) => ({ item, index }));

  result.sort((a, b) => {
    let cmp = 0;
    if (descriptor.compare) {
      cmp = descriptor.compare(a.item, b.item);
      if (order === "descend") cmp = -cmp;
    } else if (descriptor.getValue) {
      const aVal = descriptor.getValue(a.item);
      const bVal = descriptor.getValue(b.item);
      const aMissing = aVal == null || aVal === "";
      const bMissing = bVal == null || bVal === "";
      if (aMissing && bMissing) cmp = a.index - b.index;
      else if (aMissing) cmp = 1;
      // a goes after b (always sink)
      else if (bMissing) cmp = -1;
      // b goes after a (always sink)
      else {
        cmp = compareValues(aVal, bVal);
        if (order === "descend") cmp = -cmp;
      }
    }
    if (cmp === 0) cmp = a.index - b.index;
    return cmp;
  });

  return result.map((r) => r.item);
}
