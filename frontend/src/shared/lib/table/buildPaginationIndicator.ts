export interface PaginationIndicatorModel {
  left: string;
  right: string;
  fontSize: number;
}

export function buildPaginationIndicator(
  page: number,
  pageCount: number,
  width: number,
  height: number,
): PaginationIndicatorModel {
  if (pageCount <= 1 || width <= 0 || height <= 0) {
    return { left: "", right: "", fontSize: 12 };
  }

  const fontSize = Math.max(12, Math.floor(height * 0.95));
  const charWidth = Math.max(fontSize * 0.6, 6);
  const count = Math.max(2, Math.floor(width / charWidth / 2) * 2);
  const half = count / 2;

  if (page === 1) {
    return { left: "", right: ">".repeat(count), fontSize };
  }

  if (page === pageCount) {
    return { left: "<".repeat(count), right: "", fontSize };
  }

  return { left: "<".repeat(half), right: ">".repeat(half), fontSize };
}
