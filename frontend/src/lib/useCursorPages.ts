import { useCallback, useEffect, useRef, useState } from 'react';

export type CursorPage<T> = {
  items: T[];
  nextCursor: string | null;
};

// Must be stable (module-level or memoized): a new fetcher restarts the list from the first page.
export type CursorPageFetcher<T> = (
  cursor: string | null,
  signal: AbortSignal,
) => Promise<CursorPage<T>>;

// "Load more" list: loads the first page on mount, appends later pages on demand and drops
// items already shown, so overlapping pages never produce duplicate rows.
export function useCursorPages<T extends { id: string }>(fetchPage: CursorPageFetcher<T>) {
  const [items, setItems] = useState<T[]>([]);
  // Cursor for the next request; null before the first page has loaded.
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [isDone, setIsDone] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const controllerRef = useRef<AbortController | null>(null);

  const loadPage = useCallback(
    async (cursor: string | null, controller: AbortController) => {
      setIsLoading(true);
      setError(null);

      try {
        const page = await fetchPage(cursor, controller.signal);
        if (controller.signal.aborted) {
          return;
        }
        setItems((current) => {
          const seen = new Set(current.map((item) => item.id));
          return [...current, ...page.items.filter((item) => !seen.has(item.id))];
        });
        setNextCursor(page.nextCursor);
        setIsDone(page.nextCursor === null);
      } catch (loadError) {
        if (!controller.signal.aborted) {
          setError(loadError instanceof Error ? loadError.message : 'Unable to load more items.');
        }
      } finally {
        if (!controller.signal.aborted) {
          setIsLoading(false);
        }
      }
    },
    [fetchPage],
  );

  useEffect(() => {
    const controller = new AbortController();
    controllerRef.current = controller;
    setItems([]);
    setNextCursor(null);
    setIsDone(false);
    void loadPage(null, controller);

    return () => controller.abort();
  }, [loadPage]);

  // Also serves as "retry": a failed request leaves the cursor unchanged.
  const loadMore = useCallback(() => {
    const controller = controllerRef.current;
    if (!controller || isLoading || isDone) {
      return;
    }
    void loadPage(nextCursor, controller);
  }, [isDone, isLoading, loadPage, nextCursor]);

  return { items, hasMore: !isDone, isLoading, error, loadMore };
}
