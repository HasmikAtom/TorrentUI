import { useCallback, useEffect, useRef, useState } from "react";
import { fetchMovies, FetchError } from "./api";
import type { Movie, SortKey } from "./types";

const PAGE_SIZE = 50;

export type MoviesQueryState = {
  items: Movie[];
  total: number;
  loading: boolean;
  error: FetchError | null;
  done: boolean;
  sort: SortKey;
  setSort: (s: SortKey) => void;
  loadMore: () => void;
  retry: () => void;
};

export function useMoviesQuery(): MoviesQueryState {
  const [items, setItems] = useState<Movie[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<FetchError | null>(null);
  const [sort, setSortInternal] = useState<SortKey>("addedAt:desc");
  const startRef = useRef(0);
  const requestIdRef = useRef(0);

  const fetchPage = useCallback(
    async (start: number, currentSort: SortKey, replace: boolean) => {
      const myId = ++requestIdRef.current;
      setLoading(true);
      setError(null);
      const res = await fetchMovies(start, PAGE_SIZE, currentSort);
      if (myId !== requestIdRef.current) return; // a newer request superseded us
      setLoading(false);
      if (!res.ok) {
        setError(res.error);
        return;
      }
      setTotal(res.data.total);
      setItems((prev) => (replace ? res.data.items : [...prev, ...res.data.items]));
      startRef.current = start + res.data.items.length;
    },
    []
  );

  useEffect(() => {
    startRef.current = 0;
    fetchPage(0, sort, true);
  }, [sort, fetchPage]);

  const loadMore = useCallback(() => {
    if (loading) return;
    if (items.length >= total) return;
    fetchPage(startRef.current, sort, false);
  }, [loading, items.length, total, sort, fetchPage]);

  const retry = useCallback(() => {
    fetchPage(startRef.current, sort, items.length === 0);
  }, [fetchPage, sort, items.length]);

  const setSort = useCallback((s: SortKey) => {
    setItems([]);
    setTotal(0);
    setSortInternal(s);
  }, []);

  return {
    items,
    total,
    loading,
    error,
    done: items.length >= total && total > 0,
    sort,
    setSort,
    loadMore,
    retry,
  };
}
