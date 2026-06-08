import { useState, useEffect, useCallback } from 'react';

export function useFetch<T>(fetcher: () => Promise<T>, interval?: number) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetcher();
      setData(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [fetcher]);

  useEffect(() => {
    refetch();
    if (interval && interval > 0) {
      const id = setInterval(refetch, interval);
      return () => clearInterval(id);
    }
  }, [refetch, interval]);

  return { data, loading, error, refetch };
}
