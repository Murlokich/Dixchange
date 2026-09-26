import { useCallback, useEffect, useRef, useState } from 'react';

const defaultPollIntervalMs = 5000;
const minPollIntervalMs = 1000;

// Falls back to the default when unset or invalid, and never polls faster than the minimum.
export function readPollIntervalMs(configured: string | undefined) {
  const intervalMs = Number(configured);

  if (!Number.isFinite(intervalMs) || intervalMs <= 0) {
    return defaultPollIntervalMs;
  }

  return Math.max(intervalMs, minPollIntervalMs);
}

// A fetcher resolving to null means "ignore this response" and keeps the current data.
export type PollFetcher<T> = (signal: AbortSignal) => Promise<T | null>;

// Polls `fetcher` every `intervalMs`, restarting (and clearing data) whenever the fetcher changes,
// so callers must keep it stable with useCallback/useMemo. Pass null to stop polling.
export function usePolling<T>(fetcher: PollFetcher<T> | null, intervalMs: number) {
  const [data, setData] = useState<T | null>(null);
  const refreshRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    setData(null);
    if (!fetcher) {
      return;
    }

    const controller = new AbortController();
    let timer: number | undefined;
    let isFetching = false;
    let isRefreshQueued = false;

    // Each poll is scheduled only after the previous one settles, so slow responses never pile up.
    const poll = async () => {
      if (isFetching) {
        // A refresh requested mid-request (e.g. after a deposit) runs right after it instead of being lost.
        isRefreshQueued = true;
        return;
      }
      isFetching = true;
      window.clearTimeout(timer);

      try {
        const result = await fetcher(controller.signal);
        if (result !== null && !controller.signal.aborted) {
          setData(result);
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          console.error(error);
        }
      } finally {
        isFetching = false;
      }

      if (controller.signal.aborted || document.hidden) {
        return;
      }
      if (isRefreshQueued) {
        isRefreshQueued = false;
        void poll();
      } else {
        timer = window.setTimeout(poll, intervalMs);
      }
    };

    // Polling pauses while the tab is hidden and refreshes immediately when it is shown again.
    const handleVisibilityChange = () => {
      if (document.hidden) {
        window.clearTimeout(timer);
      } else {
        void poll();
      }
    };

    void poll();
    refreshRef.current = () => void poll();
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      controller.abort();
      refreshRef.current = null;
      window.clearTimeout(timer);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [fetcher, intervalMs]);

  const refresh = useCallback(() => refreshRef.current?.(), []);

  return { data, refresh };
}
