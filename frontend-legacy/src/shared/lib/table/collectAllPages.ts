export interface PaginatedFetcher<T> {
  (
    page: number,
    pageSize: number,
  ): Promise<{ items: T[]; total: number; platforms?: string[] }>;
}

export interface CollectAllPagesResult<T> {
  items: T[];
  platforms: string[];
}

export async function collectAllPages<T>(
  fetcher: PaginatedFetcher<T>,
  chunkSize = 200,
): Promise<CollectAllPagesResult<T>> {
  const first = await fetcher(1, chunkSize);
  const platforms = first.platforms ?? [];
  if (first.total <= chunkSize) return { items: first.items, platforms };

  const all: T[] = [...first.items];
  const totalPages = Math.ceil(first.total / chunkSize);

  for (let p = 2; p <= totalPages; p++) {
    const page = await fetcher(p, chunkSize);
    all.push(...page.items);
  }

  // Dedup by id if available
  const seen = new Set<number>();
  const items = all.filter((item: any) => {
    const id = item.id ?? item.memberId ?? item.ID;
    if (id == null) return true;
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  });
  return { items, platforms };
}
