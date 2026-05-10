export interface PageRange {
  start: number;
  end: number;
}

export function packMeasuredRows(
  rowHeights: number[],
  availableBodyHeight: number,
  reserveTrailingPx = 0,
): PageRange[] {
  const effectiveH = Math.max(0, availableBodyHeight - reserveTrailingPx);
  const pages: PageRange[] = [];
  if (rowHeights.length === 0) return pages;
  if (effectiveH <= 0) {
    // Can't fit anything — single row per page
    for (let i = 0; i < rowHeights.length; i++) {
      pages.push({ start: i, end: i });
    }
    return pages;
  }
  let pageStart = 0;
  let used = 0;
  for (let i = 0; i < rowHeights.length; i++) {
    if (used + rowHeights[i] > effectiveH && i > pageStart) {
      pages.push({ start: pageStart, end: i - 1 });
      pageStart = i;
      used = rowHeights[i];
    } else {
      used += rowHeights[i];
    }
  }
  pages.push({ start: pageStart, end: rowHeights.length - 1 });
  return pages;
}
