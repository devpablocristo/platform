import { useMemo, useState, useEffect } from "react";
import { search, type SearchEntry, type SearchOptions } from "@devpablocristo/core-browser/search";

/**
 * Hook genérico de búsqueda fuzzy para listas client-side.
 *
 * Convierte `items` a `SearchEntry[]` usando `textFn`, aplica debounce
 * opcional al query, y devuelve los items filtrados en orden de relevancia.
 *
 * Cuando `query` está vacío devuelve todos los items sin filtrar.
 */
export function useSearch<T>(
  items: T[],
  textFn: (item: T) => string,
  query: string,
  options?: SearchOptions & { debounceMs?: number },
): T[] {
  const { debounceMs, ...searchOptions } = options ?? {};

  // Debounce del query
  const [debouncedQuery, setDebouncedQuery] = useState(query);
  useEffect(() => {
    if (!debounceMs || debounceMs <= 0) {
      setDebouncedQuery(query);
      return;
    }
    const timer = setTimeout(() => setDebouncedQuery(query), debounceMs);
    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  const entries = useMemo<SearchEntry<T>[]>(
    () => items.map((item) => ({ item, text: textFn(item) })),
    [items, textFn],
  );

  return useMemo(() => {
    const q = debouncedQuery.trim();
    if (q.length === 0) return items;
    return search(q, entries, searchOptions).map((r) => r.item);
  }, [debouncedQuery, items, entries]);
}
