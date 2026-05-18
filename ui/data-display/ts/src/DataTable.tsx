import React, { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { PaginationBar } from "./PaginationBar";

type FilterType = "text" | "number" | "select" | "date";

export type DataTableColumn<T> = {
  key: keyof T;
  header: string;
  render?: (value: unknown, item: T) => ReactNode;
  filterable?: boolean;
  filterType?: FilterType;
  filterOptions?: string[];
  sortable?: boolean;
  width?: string;
  minWidth?: string;
  maxWidth?: string;
  align?: "left" | "center" | "right";
  padding?: "xs" | "sm" | "md";
  wrap?: boolean;
  headerAlign?: "left" | "center" | "right";
  headerPadding?: "xs" | "sm" | "md";
  headerWrap?: boolean;
};

export type DataTableProps<T> = {
  data: T[];
  filters?: Record<string, unknown>;
  onFilterChange?: (filters: Record<string, unknown>) => void;
  columns: DataTableColumn<T>[];
  headerComponent?: ReactNode;
  expandableRowRender?: (item: T) => ReactNode;
  onEdit?: (item: T) => void;
  canEdit?: (item: T) => boolean;
  onCopy?: (item: T) => void;
  onDelete?: (item: T) => void;
  renderActions?: (item: T) => ReactNode;
  actionsHeader?: string;
  className?: string;
  pagination?: {
    page: number;
    perPage: number;
    total: number;
    onPageChange: (page: number) => void;
    serverSide?: boolean;
  };
  message?: string;
  enableFilters?: boolean;
  rowStyle?: "default" | "softZebra";
};

function SortNeutralIcon() {
  return <span aria-hidden="true">⇅</span>;
}

function SortAscIcon() {
  return <span aria-hidden="true">↑</span>;
}

function SortDescIcon() {
  return <span aria-hidden="true">↓</span>;
}

function FilterIcon({ active }: { active: boolean }) {
  return <span aria-hidden="true">{active ? "⨯" : "⛃"}</span>;
}

function EditIcon() {
  return <span aria-hidden="true">✎</span>;
}

function CopyIcon() {
  return <span aria-hidden="true">⎘</span>;
}

function ArchiveIcon() {
  return <span aria-hidden="true">🗄</span>;
}

function headerAlignmentClass<T>(column: DataTableColumn<T>): string {
  if (column.headerAlign === "center") return "text-center";
  if (column.headerAlign === "right") return "text-right";
  return "text-left";
}

function cellAlignmentClass<T>(column: DataTableColumn<T>): string {
  if (column.align === "center") return "text-center";
  if (column.align === "right") return "text-right";
  return "text-left";
}

function paddingClass(size?: "xs" | "sm" | "md"): string {
  if (size === "xs") return "px-2 py-1";
  if (size === "sm") return "px-3 py-2";
  return "px-4 py-3";
}

function parseLeadingNumber(value: string): number | null {
  const match = value.trim().match(/^-?\d+(?:[.,]\d+)?/);
  if (!match) return null;

  const parsed = Number(match[0].replace(",", "."));
  return Number.isNaN(parsed) ? null : parsed;
}

function isNumericSearch(value: string): boolean {
  return /^-?\d[\d.,]*$/.test(value.trim());
}

function compareFilterOptions(a: string, b: string): number {
  const aNumber = parseLeadingNumber(a);
  const bNumber = parseLeadingNumber(b);

  if (aNumber !== null && bNumber !== null && aNumber !== bNumber) {
    return bNumber - aNumber;
  }
  if (aNumber !== null && bNumber === null) return -1;
  if (aNumber === null && bNumber !== null) return 1;

  return a.localeCompare(b, "es", {
    sensitivity: "base",
    numeric: true,
  });
}

function getVisibleFilterOptions(options: string[], search: string): string[] {
  const query = search.trim().toLowerCase();

  return [...options].sort(compareFilterOptions).filter((option) => {
    const normalizedOption = option.trim().toLowerCase();
    if (!query) return true;
    if (isNumericSearch(query)) return normalizedOption.startsWith(query);
    return normalizedOption.includes(query);
  });
}

export function DataTable<T>({
  data,
  filters,
  onFilterChange,
  columns,
  headerComponent,
  expandableRowRender,
  onEdit,
  canEdit,
  onCopy,
  onDelete,
  renderActions,
  actionsHeader = "Acciones",
  className,
  pagination,
  message = "No hay registros disponibles",
  enableFilters = false,
  rowStyle = "default",
}: DataTableProps<T>) {
  const hasActions = Boolean(onEdit || onDelete || onCopy || renderActions);
  const totalColumns = columns.length + (expandableRowRender ? 1 : 0) + (hasActions ? 1 : 0);
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<"asc" | "desc" | null>(null);
  const [expandedRow, setExpandedRow] = useState<number | null>(null);
  const [activeFilter, setActiveFilter] = useState<string | null>(null);
  const [filterSearch, setFilterSearch] = useState<Record<string, string>>({});
  const filterRefs = useRef<Record<string, HTMLDivElement | null>>({});

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        activeFilter &&
        !Object.values(filterRefs.current).some((ref) => ref && ref.contains(event.target as Node))
      ) {
        setActiveFilter(null);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [activeFilter]);

  const toggleSort = (key: string) => {
    if (sortKey !== key) {
      setSortKey(key);
      setSortDirection("asc");
      return;
    }
    if (sortDirection === "asc") {
      setSortDirection("desc");
      return;
    }
    if (sortDirection === "desc") {
      setSortKey(null);
      setSortDirection(null);
      return;
    }
    setSortDirection("asc");
  };

  const sortedData = useMemo(() => {
    if (!sortKey || !sortDirection) return data;
    const column = columns.find((item) => String(item.key) === sortKey);
    if (!column) return data;

    return [...data].sort((left, right) => {
      const aValue = left[column.key];
      const bValue = right[column.key];

      if (aValue == null && bValue == null) return 0;
      if (aValue == null) return sortDirection === "asc" ? -1 : 1;
      if (bValue == null) return sortDirection === "asc" ? 1 : -1;

      const aNum = Number(aValue);
      const bNum = Number(bValue);
      const aIsNum = !Number.isNaN(aNum) && aValue !== "";
      const bIsNum = !Number.isNaN(bNum) && bValue !== "";

      if (aIsNum && bIsNum) {
        return sortDirection === "asc" ? aNum - bNum : bNum - aNum;
      }

      return sortDirection === "asc"
        ? String(aValue).localeCompare(String(bValue))
        : String(bValue).localeCompare(String(aValue));
    });
  }, [columns, data, sortDirection, sortKey]);

  const paginatedData = useMemo(() => {
    if (!pagination) return sortedData;
    if (pagination.serverSide) return sortedData;
    const start = (pagination.page - 1) * pagination.perPage;
    const end = start + pagination.perPage;
    return sortedData.slice(start, end);
  }, [pagination, sortedData]);

  const toggleRow = (index: number) => {
    setExpandedRow((current) => (current === index ? null : index));
  };

  const toggleFilter = (key: string) => {
    setFilterSearch((prev) => ({ ...prev, [key]: prev[key] ?? "" }));
    setActiveFilter((current) => (current === key ? null : key));
  };

  const handleFilterChange = (key: string, value: unknown) => {
    onFilterChange?.({ ...(filters ?? {}), [key]: value });
  };

  const clearFilter = (key: string) => {
    const next = { ...(filters ?? {}) };
    delete next[key];
    onFilterChange?.(next);
  };

  return (
    <div
      className={`relative overflow-hidden rounded-xl border border-slate-200/80 shadow-sm ${className ?? ""}`.trim()}
    >
      {headerComponent ? <div>{headerComponent}</div> : null}

      <div className="min-h-[250px] w-full overflow-auto bg-white">
        <table className="w-full text-left text-sm text-gray-700">
          <thead className="border-b border-slate-200 bg-slate-50 text-[11px] font-semibold uppercase text-slate-500">
            <tr>
              {expandableRowRender ? <th className="w-8 p-2" /> : null}
              {columns.map((column, index) => {
                const filterValue = filters?.[String(column.key)];
                const isFilterActive = Array.isArray(filterValue)
                  ? filterValue.length > 0
                  : filterValue !== undefined && filterValue !== "";
                const isSorted = sortKey === String(column.key);

                return (
                  <th
                    key={String(column.key)}
                    className={`${headerAlignmentClass(column)} ${paddingClass(column.headerPadding)} ${
                      column.headerWrap ? "whitespace-normal break-words" : ""
                    } text-xs font-bold uppercase`}
                  >
                    <div className="inline-flex items-center gap-1">
                      <div className="flex items-center gap-1">
                        {column.sortable !== false ? (
                          <button
                            type="button"
                            onClick={() => toggleSort(String(column.key))}
                            className={`mr-1 focus:outline-none ${
                              isSorted ? "text-primary-600" : "text-slate-300 hover:text-primary-500"
                            }`}
                            title={
                              isSorted
                                ? sortDirection === "asc"
                                  ? "Orden ascendente"
                                  : "Orden descendente"
                                : "Ordenar"
                            }
                          >
                            {isSorted ? sortDirection === "asc" ? <SortAscIcon /> : <SortDescIcon /> : <SortNeutralIcon />}
                          </button>
                        ) : null}
                        {column.header}
                      </div>

                      {enableFilters && column.filterable !== false ? (
                        <div
                          className="relative"
                          ref={(element) => {
                            filterRefs.current[String(column.key)] = element;
                          }}
                        >
                          <button
                            type="button"
                            onClick={() => toggleFilter(String(column.key))}
                            className={`relative ml-1 focus:outline-none ${
                              activeFilter === String(column.key) || isFilterActive
                                ? "text-primary-500"
                                : "text-slate-300 hover:text-primary-500"
                            }`}
                            title={isFilterActive ? "Filtro activo" : "Filtrar"}
                          >
                            <FilterIcon active={activeFilter === String(column.key)} />
                            {isFilterActive ? (
                              <span className="absolute right-0 top-0 block h-2 w-2 rounded-full border border-white bg-blue-500" />
                            ) : null}
                          </button>

                          {activeFilter === String(column.key) ? (
                            <div
                              className={`absolute z-[9999] mt-2 w-48 rounded-xl border border-slate-200 bg-white p-3 shadow-lg ${
                                index === 0 ? "left-0" : "right-0"
                              }`}
                            >
                              <div className="p-2">
                                <label className="mb-1 block text-xs text-slate-600">Filtro</label>

                                {column.filterType === "select" && column.filterOptions ? (
                                  <>
                                    <input
                                      type="text"
                                      className="mb-2 w-full rounded-lg border border-slate-200 px-2.5 py-1.5 text-xs text-slate-600"
                                      placeholder="Buscar opción..."
                                      value={filterSearch[String(column.key)] || ""}
                                      onChange={(event) =>
                                        setFilterSearch((prev) => ({
                                          ...prev,
                                          [String(column.key)]: event.target.value,
                                        }))
                                      }
                                    />
                                    <div className="max-h-48 overflow-auto pr-1 text-slate-600">
                                      {getVisibleFilterOptions(
                                        column.filterOptions,
                                        filterSearch[String(column.key)] || "",
                                      ).map((option) => {
                                          const current = filters?.[String(column.key)];
                                          const selected = Array.isArray(current) ? current.includes(option) : false;

                                          return (
                                            <label
                                              key={option}
                                              className="flex items-center gap-2 py-1 text-xs text-slate-600"
                                            >
                                              <input
                                                type="checkbox"
                                                checked={selected}
                                                onChange={(event) => {
                                                  const prev = Array.isArray(current) ? current : [];
                                                  const next = event.target.checked
                                                    ? [...prev, option]
                                                    : prev.filter((value: string) => value !== option);
                                                  handleFilterChange(String(column.key), next);
                                                }}
                                              />
                                              {option}
                                            </label>
                                          );
                                        })}
                                      {getVisibleFilterOptions(
                                        column.filterOptions,
                                        filterSearch[String(column.key)] || "",
                                      ).length === 0 ? (
                                        <p className="py-1 text-xs text-slate-400">Sin resultados</p>
                                      ) : null}
                                    </div>
                                  </>
                                ) : (
                                  <input
                                    type={column.filterType === "date" ? "date" : column.filterType === "number" ? "number" : "text"}
                                    className="w-full rounded-lg border border-slate-200 px-2.5 py-1.5 text-xs text-slate-600"
                                    placeholder={column.filterType === "text" ? "Buscar..." : undefined}
                                    value={
                                      typeof filters?.[String(column.key)] === "string" ||
                                      typeof filters?.[String(column.key)] === "number"
                                        ? String(filters?.[String(column.key)])
                                        : ""
                                    }
                                    onChange={(event) => handleFilterChange(String(column.key), event.target.value)}
                                  />
                                )}

                                <div className="mt-2 flex justify-between">
                                  <button
                                    type="button"
                                    className="rounded-lg bg-slate-100 px-3 py-1.5 text-xs text-slate-600 transition-colors hover:bg-slate-200"
                                    onClick={() => clearFilter(String(column.key))}
                                  >
                                    Limpiar
                                  </button>
                                  <button
                                    type="button"
                                    className="rounded-lg bg-primary-600 px-3 py-1.5 text-xs text-white transition-colors hover:bg-primary-700"
                                    onClick={() => setActiveFilter(null)}
                                  >
                                    Aplicar
                                  </button>
                                </div>
                              </div>
                            </div>
                          ) : null}
                        </div>
                      ) : null}
                    </div>
                  </th>
                );
              })}
              {hasActions ? <th className="p-4 text-center text-xs font-bold uppercase">{actionsHeader}</th> : null}
            </tr>
          </thead>

          <tbody>
            {paginatedData.length > 0 ? (
              paginatedData.map((item, index) => {
                const zebraClass =
                  rowStyle === "softZebra"
                    ? index % 2 === 0
                      ? "bg-white hover:bg-slate-50"
                      : "bg-[#f9f9f9b8] hover:bg-slate-200"
                    : index % 2 === 0
                      ? "bg-white hover:bg-slate-50"
                      : "bg-slate-50/50 hover:bg-slate-50";

                return (
                  <React.Fragment key={index}>
                    <tr className={`border-t border-slate-100 text-slate-700 transition-colors ${zebraClass}`}>
                      {expandableRowRender ? (
                        <td
                          className="cursor-pointer px-4 py-3"
                          onClick={(event) => {
                            event.stopPropagation();
                            toggleRow(index);
                          }}
                        >
                          <span
                            className={`inline-block transition-transform duration-200 ${
                              expandedRow === index ? "rotate-90" : ""
                            }`}
                          >
                            ▶
                          </span>
                        </td>
                      ) : null}

                      {columns.map((column) => (
                        <td
                          key={String(column.key)}
                          className={`${cellAlignmentClass(column)} ${paddingClass(column.padding)} ${
                            column.wrap ? "whitespace-normal break-words" : "whitespace-nowrap truncate"
                          }`}
                          style={{
                            width: column.width,
                            minWidth: column.minWidth ?? "100px",
                            maxWidth: column.maxWidth ?? "180px",
                          }}
                          title={!column.wrap ? String(item[column.key] ?? "") : undefined}
                        >
                          {column.render ? column.render(item[column.key], item) : String(item[column.key] ?? "")}
                        </td>
                      ))}

                      {hasActions ? (
                        <td className="px-6 py-4 text-center">
                          <div className="flex justify-center space-x-2">
                            {renderActions?.(item)}
                            {onEdit ? (
                              <button
                                type="button"
                                onClick={() => {
                                  if ((canEdit?.(item) ?? true) === false) return;
                                  onEdit(item);
                                }}
                                disabled={(canEdit?.(item) ?? true) === false}
                                className={`mr-3 font-medium ${
                                  (canEdit?.(item) ?? true) === false
                                    ? "cursor-not-allowed text-slate-300"
                                    : "text-slate-500 hover:text-slate-700 hover:underline"
                                }`}
                                title={(canEdit?.(item) ?? true) === false ? "Edición bloqueada" : "Editar"}
                              >
                                <EditIcon />
                              </button>
                            ) : null}
                            {onCopy ? (
                              <button
                                type="button"
                                onClick={() => onCopy(item)}
                                className="flex items-center gap-1 text-primary-500 hover:text-primary-700"
                                title="Duplicar"
                              >
                                <CopyIcon />
                              </button>
                            ) : null}
                            {onDelete ? (
                              <button
                                type="button"
                                onClick={() => onDelete(item)}
                                className="font-medium text-amber-500 hover:text-amber-600 hover:underline"
                                title="Archivar"
                              >
                                <ArchiveIcon />
                              </button>
                            ) : null}
                          </div>
                        </td>
                      ) : null}
                    </tr>

                    {expandableRowRender && expandedRow === index ? (
                      <tr className="bg-white">
                        <td colSpan={totalColumns}>
                          <div className="p-4">{expandableRowRender(item)}</div>
                        </td>
                      </tr>
                    ) : null}
                  </React.Fragment>
                );
              })
            ) : (
              <tr>
                <td colSpan={totalColumns}>
                  <div className="p-4 py-12 text-center text-sm text-slate-500">{message}</div>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="sticky bottom-0 z-10 border-t border-slate-200 bg-white">
        {pagination && sortedData.length > 0 ? (
          <PaginationBar
            page={pagination.page}
            perPage={pagination.perPage}
            total={pagination.total}
            onPageChange={pagination.onPageChange}
          />
        ) : null}
      </div>
    </div>
  );
}

export default DataTable;
