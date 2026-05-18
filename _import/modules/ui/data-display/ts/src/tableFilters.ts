export type TableFilters = Record<string, unknown>;

export type TableFilterMatchMode = "exact" | "includes";

export type TableFilterValueGetter<T> = (row: T, key: keyof T) => unknown;

export type TableFilterConfig<T> = {
  getValue?: TableFilterValueGetter<T>;
  matchModeByColumn?: Partial<Record<keyof T, TableFilterMatchMode>>;
};

export type GetTableFilterOptionsConfig<T> = TableFilterConfig<T> & {
  sort?: (a: string, b: string) => number;
};

export function isEmptyTableFilterValue(value: unknown) {
  return (
    value === undefined ||
    value === null ||
    value === "" ||
    (Array.isArray(value) && value.length === 0)
  );
}

export function normalizeTableFilterValue(value: unknown) {
  return String(value ?? "").trim().toLowerCase();
}

function getRowValue<T>(
  row: T,
  key: keyof T,
  getValue?: TableFilterValueGetter<T>
) {
  return getValue ? getValue(row, key) : row[key];
}

export function matchesTableFilterValue(
  rowValue: unknown,
  filterValue: unknown,
  matchMode: TableFilterMatchMode = "includes"
) {
  if (isEmptyTableFilterValue(filterValue)) return true;

  const normalizedRowValue = normalizeTableFilterValue(rowValue);

  if (Array.isArray(filterValue)) {
    return filterValue.some(
      (value) => normalizedRowValue === normalizeTableFilterValue(value)
    );
  }

  const normalizedFilterValue = normalizeTableFilterValue(filterValue);
  if (matchMode === "exact") {
    return normalizedRowValue === normalizedFilterValue;
  }

  return normalizedRowValue.includes(normalizedFilterValue);
}

export function applyTableFilters<T>(
  rows: T[],
  filters: TableFilters,
  config: TableFilterConfig<T> = {},
  excludeKey?: keyof T
) {
  return rows.filter((row) =>
    Object.entries(filters).every(([rawKey, filterValue]) => {
      const key = rawKey as keyof T;
      if (key === excludeKey || isEmptyTableFilterValue(filterValue)) {
        return true;
      }

      const matchMode = config.matchModeByColumn?.[key] ?? "includes";
      return matchesTableFilterValue(
        getRowValue(row, key, config.getValue),
        filterValue,
        matchMode
      );
    })
  );
}

export function getTableFilterOptions<T>(
  rows: T[],
  filters: TableFilters,
  key: keyof T,
  config: GetTableFilterOptionsConfig<T> = {}
) {
  const filteredRows = applyTableFilters(rows, filters, config, key);
  const options = filteredRows
    .map((row) => String(getRowValue(row, key, config.getValue) ?? "").trim())
    .filter(Boolean);

  return [...new Set(options)].sort(
    config.sort ??
      ((a, b) =>
        a.localeCompare(b, "es", { sensitivity: "base", numeric: true }))
  );
}
