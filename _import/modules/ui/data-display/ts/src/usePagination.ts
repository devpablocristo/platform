import { useCallback, useMemo, useState } from "react";

export type UsePaginationOptions = {
  initialPage?: number;
  perPage?: number;
};

export type BuildPaginationOptions = {
  serverSide?: boolean;
};

export function getTotalPages(total: number, perPage: number) {
  if (perPage <= 0) return 1;
  return Math.max(1, Math.ceil(Math.max(0, total) / perPage));
}

export function usePagination({ initialPage = 1, perPage = 10 }: UsePaginationOptions = {}) {
  const normalizedPerPage = Math.max(1, perPage);
  const [page, setPageState] = useState(Math.max(1, initialPage));

  const setPage = useCallback((nextPage: number) => {
    setPageState(Math.max(1, nextPage));
  }, []);

  const resetPage = useCallback(() => {
    setPageState(1);
  }, []);

  const clampPageForTotal = useCallback(
    (total: number) => {
      setPageState((current) => Math.min(Math.max(1, current), getTotalPages(total, normalizedPerPage)));
    },
    [normalizedPerPage]
  );

  const buildPagination = useCallback(
    (total: number, options: BuildPaginationOptions = {}) => ({
      page,
      perPage: normalizedPerPage,
      total,
      onPageChange: setPage,
      serverSide: options.serverSide,
    }),
    [normalizedPerPage, page, setPage]
  );

  const queryParams = useMemo(
    () => ({
      page: String(page),
      per_page: String(normalizedPerPage),
    }),
    [normalizedPerPage, page]
  );

  return {
    page,
    perPage: normalizedPerPage,
    setPage,
    resetPage,
    clampPageForTotal,
    buildPagination,
    queryParams,
  };
}
