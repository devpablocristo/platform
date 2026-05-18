import { useCallback, useMemo, useState } from "react";

import {
  applyTableFilters,
  getTableFilterOptions,
  type GetTableFilterOptionsConfig,
  type TableFilterConfig,
  type TableFilters,
} from "./tableFilters";

export type UseClientTableFiltersOptions<T> = TableFilterConfig<T> & {
  rows: T[];
  initialFilters?: TableFilters;
  onChange?: (filters: TableFilters) => void;
};

export function useClientTableFilters<T>({
  rows,
  initialFilters = {},
  onChange,
  ...filterConfig
}: UseClientTableFiltersOptions<T>) {
  const [filters, setFilters] = useState<TableFilters>(initialFilters);
  const stableFilterConfig = useMemo(
    () => filterConfig,
    [filterConfig.getValue, filterConfig.matchModeByColumn]
  );

  const handleFilterChange = useCallback(
    (nextFilters: TableFilters) => {
      setFilters(nextFilters);
      onChange?.(nextFilters);
    },
    [onChange]
  );

  const resetFilters = useCallback(() => {
    setFilters({});
    onChange?.({});
  }, [onChange]);

  const filteredRows = useMemo(
    () => applyTableFilters(rows, filters, stableFilterConfig),
    [filters, rows, stableFilterConfig]
  );

  const getFilterOptionsForColumn = useCallback(
    (
      key: keyof T,
      optionsConfig: Omit<GetTableFilterOptionsConfig<T>, keyof TableFilterConfig<T>> = {}
    ) =>
      getTableFilterOptions(rows, filters, key, {
        ...stableFilterConfig,
        ...optionsConfig,
      }),
    [filters, rows, stableFilterConfig]
  );

  return {
    filters,
    filteredRows,
    getFilterOptionsForColumn,
    handleFilterChange,
    resetFilters,
  };
}
