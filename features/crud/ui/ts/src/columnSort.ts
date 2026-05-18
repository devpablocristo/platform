export type SortDirection = "asc" | "desc";

export function getComparableFromRow<T extends { id: string }>(
  row: T,
  key: keyof T & string,
  sortValue?: (row: T) => unknown,
): unknown {
  if (sortValue) return sortValue(row);
  return (row as Record<string, unknown>)[key];
}

/**
 * Comparación estable para valores de celda: números, booleanos, fechas ISO, texto (localeCompare numérico).
 */
export function compareUnknown(a: unknown, b: unknown, direction: SortDirection): number {
  const mul = direction === "asc" ? 1 : -1;
  const rankEmpty = (v: unknown): number => {
    if (v === null || v === undefined) return 2;
    if (typeof v === "string" && v.trim() === "") return 1;
    return 0;
  };
  const ra = rankEmpty(a);
  const rb = rankEmpty(b);
  if (ra !== rb) return (ra - rb) * mul;
  if (ra === 2 && rb === 2) return 0;

  if (typeof a === "number" && typeof b === "number" && !Number.isNaN(a) && !Number.isNaN(b)) {
    return (a - b) * mul;
  }
  if (typeof a === "boolean" && typeof b === "boolean") {
    return ((a ? 1 : 0) - (b ? 1 : 0)) * mul;
  }
  if (a instanceof Date && b instanceof Date) {
    return (a.getTime() - b.getTime()) * mul;
  }
  const sa = String(a);
  const sb = String(b);
  if (/^\d{4}-\d{2}-\d{2}/.test(sa) && /^\d{4}-\d{2}-\d{2}/.test(sb)) {
    const da = Date.parse(sa);
    const db = Date.parse(sb);
    if (!Number.isNaN(da) && !Number.isNaN(db)) return (da - db) * mul;
  }
  return sa.localeCompare(sb, undefined, { numeric: true, sensitivity: "base" }) * mul;
}
