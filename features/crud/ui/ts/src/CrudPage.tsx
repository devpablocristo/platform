/**
 * Página CRUD para consolas administrativas.
 *
 * Responsabilidad: orquestar lista, formulario y acciones sobre datos inyectados (`dataSource` o
 * `basePath` + `httpClient`). No contiene reglas de negocio ni llamadas acopladas a un producto.
 *
 * Shell de layout: `platform/browser/ts`. Orquestación CRUD: `platform/features/crud/ui/ts`.
 */
import { FormEvent, type ReactElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { CrudPageShell, parsePaginatedResponse } from "@devpablocristo/platform-browser/crud";
import { CrudShellHeaderActionsColumn } from "./CrudShellHeaderActionsColumn";
import { search as fuzzySearch, type SearchEntry } from "@devpablocristo/platform-browser/search";
import { compareUnknown, getComparableFromRow, type SortDirection } from "./columnSort";
import { crudItemPath, crudListPath } from "./restPaths";
import { interpolate, mergeCrudStrings, type CrudStrings, defaultCrudStrings } from "./strings";
import type {
  CrudColumn,
  CrudFormValues,
  CrudLifecycleView,
  CrudPageConfig,
  CrudRowAction,
  CrudToolbarAction,
} from "./types";

/**
 * Props extendidas: textos UI genéricos (fusión sobre `stringsBase`).
 */
export type CrudPageProps<T extends { id: string }> = CrudPageConfig<T> & {
  strings?: Partial<CrudStrings>;
  stringsBase?: CrudStrings;
};

function buttonClass(kind: "primary" | "secondary" | "danger" | "success" = "secondary", small = true): string {
  const size = small ? "btn-sm " : "";
  switch (kind) {
    case "primary":
      return `${size}btn-primary`;
    case "danger":
      return `${size}btn-danger`;
    case "success":
      return `${size}btn-success`;
    default:
      return `${size}btn-secondary`;
  }
}

function normalizeError(error: unknown): string {
  if (error instanceof Error) return error.message;
  return String(error);
}

export function CrudPage<T extends { id: string }>(props: CrudPageProps<T>): ReactElement {
  const {
    basePath,
    listQuery,
    paginationCursorParam = "cursor",
    dataSource,
    httpClient: httpClientProp,
    supportsArchived = false,
    supportsTrash = false,
    allowCreate,
    allowEdit,
    allowArchive,
    allowTrash,
    allowUnarchive,
    allowRestore,
    allowPurge,
    label,
    labelPlural,
    labelPluralCap,
    columns,
    archivedColumns,
    trashColumns,
    formFields,
    searchText,
    toFormValues,
    toBody,
    isValid,
    searchPlaceholder,
    emptyState,
    archivedEmptyState,
    trashEmptyState,
    createLabel,
    toolbarActions = [],
    rowActions = [],
    formatFieldText = (s) => s,
    sentenceCase = (s) => s,
    strings: stringsPartial,
    stringsBase = defaultCrudStrings,
    onExternalEdit,
    onMutationSuccess,
    onRowClick,
    preSearchFilter,
    listHeaderInlineSlot,
    externalSearch,
    viewModes: _viewModes,
    featureFlags,
    initialView = "active",
  } = props;

  const str = useMemo(() => mergeCrudStrings(stringsBase, stringsPartial), [stringsBase, stringsPartial]);
  const httpClient = httpClientProp;
  const columnSortEnabled = featureFlags?.columnSort !== false;

  const vars = useMemo(
    () => ({
      label,
      labelPlural,
      labelPluralCap,
    }),
    [label, labelPlural, labelPluralCap],
  );

  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [internalSearch, setInternalSearch] = useState("");
  const search = externalSearch ?? internalSearch;
  const [lifecycleView, setLifecycleView] = useState<CrudLifecycleView>(() => {
    if (initialView === "archived" && supportsArchived) return "archived";
    if (initialView === "trash" && supportsTrash) return "trash";
    return "active";
  });

  const [editing, setEditing] = useState<T | null>(null);
  const [creating, setCreating] = useState(false);
  const [formValues, setFormValues] = useState<CrudFormValues>({});
  const [saving, setSaving] = useState(false);

  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [confirmDeleteText, setConfirmDeleteText] = useState("");
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<SortDirection>("asc");

  // Evita condiciones de carrera (p. ej. React StrictMode doble mount) que dejan loading en true.
  const loadSeqRef = useRef(0);

  const emptyValues = Object.fromEntries(
    formFields.map((field) => [field.key, field.type === "checkbox" ? false : ""]),
  ) as CrudFormValues;
  const activeFormFields = formFields.filter((field) => {
    if (editing && field.createOnly) return false;
    if (!editing && field.editOnly) return false;
    return true;
  });

  const canCreate = allowCreate ?? (formFields.length > 0 && Boolean(dataSource?.create || basePath));
  const canEdit =
    allowEdit ??
    (Boolean(onExternalEdit) || (formFields.length > 0 && Boolean(dataSource?.update || basePath)));
  const canArchive = allowArchive ?? (supportsArchived && Boolean(dataSource?.archive || basePath));
  const canTrash = allowTrash ?? Boolean(dataSource?.trash || basePath);
  const canUnarchive = allowUnarchive ?? (supportsArchived && Boolean(dataSource?.unarchive || basePath));
  const canRestore = allowRestore ?? (supportsTrash && Boolean(dataSource?.restore || basePath));
  const canPurge = allowPurge ?? (supportsTrash && Boolean(dataSource?.purge || basePath));
  const showForm = (creating || (editing !== null && !onExternalEdit)) && formFields.length > 0;
  const purgeWord = str.confirmWord;

  const showActionsColumn =
    lifecycleView === "archived"
      ? canUnarchive
      : lifecycleView === "trash"
        ? canRestore || canPurge
        : canEdit || rowActions.length > 0 || canArchive || canTrash;
  const visibleColumns =
    lifecycleView === "archived" && archivedColumns?.length
      ? archivedColumns
      : lifecycleView === "trash" && trashColumns?.length
        ? trashColumns
        : columns;
  const rowClickHandler = lifecycleView === "active" ? onRowClick : undefined;

  const defaultPageSize = 100;

  function buildListPath(cursor?: string): string {
    let path = crudListPath(basePath!, lifecycleView);
    const params: string[] = [];
    if (listQuery) params.push(listQuery);
    params.push(`limit=${defaultPageSize}`);
    if (cursor) params.push(`${encodeURIComponent(paginationCursorParam)}=${encodeURIComponent(cursor)}`);
    if (params.length > 0) {
      path = path.includes("?") ? `${path}&${params.join("&")}` : `${path}?${params.join("&")}`;
    }
    return path;
  }

  async function loadItems(): Promise<void> {
    const seq = ++loadSeqRef.current;
    setLoading(true);
    setError("");
    setHasMore(false);
    setNextCursor(null);
    try {
      if (dataSource?.list) {
        const rows = await dataSource.list({ view: lifecycleView });
        if (seq !== loadSeqRef.current) return;
        setItems(rows);
        return;
      }
      if (!basePath || !httpClient) {
        if (seq !== loadSeqRef.current) return;
        if (!basePath) { setItems([]); return; }
        setError("CrudPage: basePath requires httpClient or dataSource.list");
        setItems([]);
        return;
      }
      const data = await httpClient.json<unknown>(buildListPath());
      if (seq !== loadSeqRef.current) return;
      const page = parsePaginatedResponse<T>(data);
      setItems(page.items);
      setHasMore(page.hasMore);
      setNextCursor(page.nextCursor || null);
    } catch (err) {
      if (seq === loadSeqRef.current) setError(normalizeError(err));
    } finally {
      if (seq === loadSeqRef.current) setLoading(false);
    }
  }

  async function loadMore(): Promise<void> {
    if (!basePath || !httpClient || !nextCursor) return;
    setLoadingMore(true);
    try {
      const data = await httpClient.json<unknown>(buildListPath(nextCursor));
      const page = parsePaginatedResponse<T>(data);
      setItems((prev) => [...prev, ...page.items]);
      setHasMore(page.hasMore);
      setNextCursor(page.nextCursor || null);
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setLoadingMore(false);
    }
  }

  useEffect(() => {
    void loadItems();
  }, [lifecycleView]);

  function closeForm(): void {
    setCreating(false);
    setEditing(null);
    setFormValues({});
  }

  function openCreate(): void {
    setEditing(null);
    setCreating(true);
    setFormValues({ ...emptyValues });
  }

  function openEdit(row: T): void {
    setCreating(false);
    setEditing(row);
    setFormValues(toFormValues(row));
  }

  function cancelPurge(): void {
    setConfirmDeleteId(null);
    setConfirmDeleteText("");
  }

  function setField(key: string, value: string | boolean): void {
    setFormValues((current) => ({ ...current, [key]: value }));
  }

  async function submitForm(event: FormEvent): Promise<void> {
    event.preventDefault();
    if (!isValid(formValues)) return;

    setSaving(true);
    setError("");
    try {
      if (editing) {
        if (dataSource?.update) {
          await dataSource.update(editing, formValues);
        } else if (basePath && httpClient) {
          await httpClient.json(crudItemPath(basePath, editing.id), { method: "PUT", body: toBody ? toBody(formValues) : {} });
        }
      } else if (dataSource?.create) {
        await dataSource.create(formValues);
      } else if (basePath && httpClient) {
        await httpClient.json(basePath, { method: "POST", body: toBody ? toBody(formValues) : {} });
      }
      closeForm();
      await loadItems();
      await onMutationSuccess?.({ action: editing ? "update" : "create", row: editing ?? undefined });
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setSaving(false);
    }
  }

  async function archiveRow(row: T): Promise<void> {
    const nextBusyKey = `${row.id}:archive`;
    setBusyKey(nextBusyKey);
    setError("");
    try {
      if (dataSource?.archive) {
        await dataSource.archive(row);
      } else if (basePath && httpClient) {
        await httpClient.json(crudItemPath(basePath, row.id, "archive"), { method: "POST", body: {} });
      }
      await loadItems();
      await onMutationSuccess?.({ action: "archive", row });
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setBusyKey(null);
    }
  }

  async function trashRow(row: T): Promise<void> {
    const nextBusyKey = `${row.id}:trash`;
    setBusyKey(nextBusyKey);
    setError("");
    try {
      if (dataSource?.trash) {
        await dataSource.trash(row);
      } else if (basePath && httpClient) {
        await httpClient.json(crudItemPath(basePath, row.id, "trash"), { method: "POST", body: {} });
      }
      await loadItems();
      await onMutationSuccess?.({ action: "trash", row });
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setBusyKey(null);
    }
  }

  async function unarchiveRow(row: T): Promise<void> {
    const nextBusyKey = `${row.id}:unarchive`;
    setBusyKey(nextBusyKey);
    setError("");
    try {
      if (dataSource?.unarchive) {
        await dataSource.unarchive(row);
      } else if (basePath && httpClient) {
        await httpClient.json(crudItemPath(basePath, row.id, "unarchive"), { method: "POST", body: {} });
      }
      await loadItems();
      await onMutationSuccess?.({ action: "unarchive", row });
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setBusyKey(null);
    }
  }

  async function restoreRow(row: T): Promise<void> {
    const nextBusyKey = `${row.id}:restore`;
    setBusyKey(nextBusyKey);
    setError("");
    try {
      if (dataSource?.restore) {
        await dataSource.restore(row);
      } else if (basePath && httpClient) {
        await httpClient.json(crudItemPath(basePath, row.id, "restore"), { method: "POST", body: {} });
      }
      await loadItems();
      await onMutationSuccess?.({ action: "restore", row });
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setBusyKey(null);
    }
  }

  async function purgeRow(row: T): Promise<void> {
    const nextBusyKey = `${row.id}:purge`;
    setBusyKey(nextBusyKey);
    setError("");
    try {
      if (dataSource?.purge) {
        await dataSource.purge(row);
      } else if (basePath && httpClient) {
        await httpClient.json(crudItemPath(basePath, row.id, "purge"), { method: "DELETE" });
      }
      cancelPurge();
      await loadItems();
      await onMutationSuccess?.({ action: "purge", row });
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setBusyKey(null);
    }
  }

  async function runToolbarAction(action: CrudToolbarAction<T>): Promise<void> {
    setError("");
    try {
      await action.onClick({
        items,
        reload: loadItems,
        setError,
      });
    } catch (err) {
      setError(normalizeError(err));
    }
  }

  async function runRowAction(action: CrudRowAction<T>, row: T): Promise<void> {
    const nextBusyKey = `${row.id}:${action.id}`;
    setBusyKey(nextBusyKey);
    setError("");
    try {
      await action.onClick(row, {
        items,
        reload: loadItems,
        setError,
      });
    } catch (err) {
      setError(normalizeError(err));
    } finally {
      setBusyKey(null);
    }
  }

  const preSearchItems = useMemo(() => {
    if (!preSearchFilter) return items;
    return preSearchFilter(items);
  }, [items, preSearchFilter]);

  const searchEntries = useMemo<SearchEntry<T>[]>(
    () => preSearchItems.map((row) => ({ item: row, text: searchText(row) })),
    [preSearchItems, searchText],
  );

  const filtered = useMemo(() => {
    const q = search.trim();
    if (q.length === 0) return preSearchItems;
    return fuzzySearch(q, searchEntries).map((r) => r.item);
  }, [search, preSearchItems, searchEntries]);

  const sortedRows = useMemo(() => {
    if (!columnSortEnabled || sortKey == null) return filtered;
    const col = visibleColumns.find((c) => c.key === sortKey) as CrudColumn<T> | undefined;
    if (!col || col.sortable === false) return filtered;
    const key = col.key;
    const sv = col.sortValue;
    return [...filtered].sort((ra, rb) =>
      compareUnknown(getComparableFromRow(ra, key, sv), getComparableFromRow(rb, key, sv), sortDir),
    );
  }, [columnSortEnabled, filtered, sortKey, sortDir, visibleColumns]);

  const onColumnSortClick = useCallback(
    (column: CrudColumn<T>) => {
      if (!columnSortEnabled || column.sortable === false) return;
      const k = column.key;
      if (sortKey !== k) {
        setSortKey(k);
        setSortDir("asc");
      } else {
        setSortDir((d) => (d === "asc" ? "desc" : "asc"));
      }
    },
    [columnSortEnabled, sortKey],
  );

  const visibleToolbarActions = toolbarActions.filter((action) => action.isVisible?.({ view: lifecycleView, items }) ?? true);
  const rawToolbarActionCount = toolbarActions?.length ?? 0;
  /** Incluye acciones declaradas aunque `isVisible` las oculte en este momento (evita fila vacía sin CSV/archivados). */
  const showToolbarButtonRow =
    visibleToolbarActions.length > 0 || canCreate || supportsArchived || supportsTrash || rawToolbarActionCount > 0;

  const searchPlaceholderResolved =
    searchPlaceholder != null && searchPlaceholder !== ""
      ? formatFieldText(searchPlaceholder)
      : interpolate(str.searchPlaceholder, vars);

  const titleActive = sentenceCase(labelPluralCap);
  const titleArchivedView = sentenceCase(interpolate(str.titleArchived, { ...vars, labelPluralCap }));
  const titleTrashView = sentenceCase(interpolate(str.titleTrash, { ...vars, labelPluralCap }));

  const shellSearchField =
    externalSearch == null && featureFlags?.searchBar !== false
      ? {
          value: internalSearch,
          onChange: setInternalSearch,
          placeholder: searchPlaceholderResolved,
          inputClassName: "m-kanban__search",
        }
      : null;

  const toolbarButtonGroup = showToolbarButtonRow ? (
    <>
      {visibleToolbarActions.map((action) => (
        <button
          key={action.id}
          type="button"
          className={buttonClass(action.kind)}
          onClick={() => {
            void runToolbarAction(action);
          }}
        >
          {formatFieldText(action.label)}
        </button>
      ))}
      {canCreate && (
        <button type="button" className="btn-sm btn-primary" onClick={openCreate}>
          {createLabel ? formatFieldText(createLabel) : sentenceCase(interpolate(str.buttonNew, vars))}
        </button>
      )}
      {supportsArchived && (
        <button
          type="button"
          className={`btn-sm ${lifecycleView === "archived" ? "btn-primary" : "btn-secondary"}`}
          onClick={() => {
            closeForm();
            cancelPurge();
            setLifecycleView((current) => current === "archived" ? "active" : "archived");
          }}
        >
          {lifecycleView === "archived" ? str.toggleShowActive : str.toggleShowArchived}
        </button>
      )}
      {supportsTrash && (
        <button
          type="button"
          className={`btn-sm ${lifecycleView === "trash" ? "btn-primary" : "btn-secondary"}`}
          onClick={() => {
            closeForm();
            cancelPurge();
            setLifecycleView((current) => current === "trash" ? "active" : "trash");
          }}
        >
          {lifecycleView === "trash" ? str.toggleShowActive : str.toggleShowTrash}
        </button>
      )}
    </>
  ) : null;

  const headerActionsResolved =
    shellSearchField || toolbarButtonGroup ? (
      <CrudShellHeaderActionsColumn search={shellSearchField}>{toolbarButtonGroup}</CrudShellHeaderActionsColumn>
    ) : undefined;

  return (
    <CrudPageShell
      title={lifecycleView === "archived" ? titleArchivedView : lifecycleView === "trash" ? titleTrashView : titleActive}
      subtitle={
        loading
          ? str.statusLoading
          : `${sortedRows.length} ${sortedRows.length === 1 ? label : labelPlural}`
      }
      headerLeadSlot={
        listHeaderInlineSlot != null && featureFlags?.headerQuickFilterStrip !== false ? (
          <div className="crud-list-header-lead">{listHeaderInlineSlot({ items })}</div>
        ) : undefined
      }
      headerActions={headerActionsResolved}
      error={error ? <div className="alert alert-error">{error}</div> : undefined}
      form={
        showForm && (lifecycleView === "active" || creating) ? (
          <div className="card crud-form-card">
            <div className="card-header">
              <h2>
                {sentenceCase(
                  interpolate(editing ? str.formEdit : str.formCreate, vars),
                )}
              </h2>
            </div>
            <form
              onSubmit={(event) => {
                void submitForm(event);
              }}
              className="crud-form"
            >
              <div className="crud-form-grid">
                {activeFormFields.map((field) => (
                  <div key={field.key} className={`form-group${field.fullWidth ? " full-width" : ""}`}>
                    <label htmlFor={`crud-field-${field.key}`}>
                      {formatFieldText(field.label)}
                      {field.required ? " *" : ""}
                    </label>
                    {field.type === "textarea" ? (
                      <textarea
                        id={`crud-field-${field.key}`}
                        rows={3}
                        value={String(formValues[field.key] ?? "")}
                        onChange={(event) => setField(field.key, event.target.value)}
                        placeholder={field.placeholder ? formatFieldText(field.placeholder) : undefined}
                      />
                    ) : field.type === "select" ? (
                      <select
                        id={`crud-field-${field.key}`}
                        value={String(formValues[field.key] ?? "")}
                        onChange={(event) => setField(field.key, event.target.value)}
                      >
                        <option value="">{field.placeholder ? formatFieldText(field.placeholder) : str.selectPlaceholder}</option>
                        {(field.options ?? []).map((option) => (
                          <option key={option.value} value={option.value}>
                            {formatFieldText(option.label)}
                          </option>
                        ))}
                      </select>
                    ) : field.type === "checkbox" ? (
                      <label className="toggle">
                        <input
                          id={`crud-field-${field.key}`}
                          aria-label={formatFieldText(field.label)}
                          type="checkbox"
                          checked={Boolean(formValues[field.key])}
                          onChange={(event) => setField(field.key, event.target.checked)}
                        />
                        <span className="toggle-track" />
                        <span className="toggle-thumb" />
                      </label>
                    ) : (
                      <input
                        id={`crud-field-${field.key}`}
                        type={field.type ?? "text"}
                        value={String(formValues[field.key] ?? "")}
                        onChange={(event) => setField(field.key, event.target.value)}
                        placeholder={field.placeholder ? formatFieldText(field.placeholder) : undefined}
                        autoFocus={field === activeFormFields[0]}
                      />
                    )}
                  </div>
                ))}
              </div>
              <div className="actions-row">
                <button type="submit" className="btn-primary" disabled={saving || !isValid(formValues)}>
                  {saving ? str.statusSaving : str.actionSave}
                </button>
                <button type="button" className="btn-secondary" onClick={closeForm} disabled={saving}>
                  {str.actionCancel}
                </button>
              </div>
            </form>
          </div>
        ) : undefined
      }
    >
      {loading ? (
        <div className="spinner" />
      ) : sortedRows.length === 0 ? (
        <div className="empty-state">
          <p>
            {search.trim()
              ? interpolate(str.emptySearch, { ...vars, search: search.trim() })
              : lifecycleView === "archived"
                ? archivedEmptyState
                  ? formatFieldText(archivedEmptyState)
                  : interpolate(str.emptyArchived, vars)
                : lifecycleView === "trash"
                  ? trashEmptyState
                    ? formatFieldText(trashEmptyState)
                    : interpolate(str.emptyTrash, vars)
                : emptyState
                  ? formatFieldText(emptyState)
                  : interpolate(str.emptyActive, vars)}
          </p>
          {!search.trim() && canCreate && (
            <button type="button" className="btn-primary" onClick={openCreate}>
              {createLabel
                ? formatFieldText(createLabel)
                : sentenceCase(interpolate(str.buttonCreateFirst, vars))}
            </button>
          )}
        </div>
      ) : (
        <div className="table-wrap">
          <table className="crud-table">
            <thead>
              <tr>
                {visibleColumns.map((column) => {
                  const sortable = columnSortEnabled && column.sortable !== false;
                  const active = sortKey === column.key;
                  const ariaSort = !sortable ? undefined : active ? (sortDir === "asc" ? "ascending" : "descending") : "none";
                  const labelText = sentenceCase(formatFieldText(column.header));
                  return (
                    <th key={column.key} className={column.className} aria-sort={ariaSort}>
                      {sortable ? (
                        <button
                          type="button"
                          className="crud-table__sort-btn"
                          onClick={() => onColumnSortClick(column)}
                          aria-label={`Ordenar por ${labelText}`}
                        >
                          <span className="crud-table__sort-label">{labelText}</span>
                          <span className="crud-table__sort-icons" aria-hidden>
                            <span className={active && sortDir === "asc" ? "crud-table__sort-icon crud-table__sort-icon--active" : "crud-table__sort-icon"}>
                              ▲
                            </span>
                            <span className={active && sortDir === "desc" ? "crud-table__sort-icon crud-table__sort-icon--active" : "crud-table__sort-icon"}>
                              ▼
                            </span>
                          </span>
                        </button>
                      ) : (
                        labelText
                      )}
                    </th>
                  );
                })}
                {showActionsColumn ? (
                  <th className="col-actions">{sentenceCase(str.tableActions)}</th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {sortedRows.map((row) => {
                const visibleRowActions = rowActions.filter(
                  (action) => action.isVisible?.(row, { view: lifecycleView }) ?? true,
                );
                return (
                  <tr
                    key={row.id}
                    className={rowClickHandler ? "crud-table__row-clickable" : undefined}
                    onClick={rowClickHandler ? () => { rowClickHandler(row); } : undefined}
                  >
                    {visibleColumns.map((column) => (
                      <td key={column.key} className={column.className}>
                        {column.render ? column.render(row[column.key], row) : String(row[column.key] ?? "") || "---"}
                      </td>
                    ))}
                    {showActionsColumn ? (
                    <td
                      className="col-actions"
                      onClick={rowClickHandler ? (event) => { event.stopPropagation(); } : undefined}
                    >
                      {lifecycleView === "archived" ? (
                        <div className="crud-row-actions">
                          {canUnarchive && (
                            <button
                              type="button"
                              className="btn-sm btn-primary"
                              disabled={busyKey === `${row.id}:unarchive`}
                              onClick={() => {
                                void unarchiveRow(row);
                              }}
                            >
                              {busyKey === `${row.id}:unarchive` ? "..." : str.actionUnarchive}
                            </button>
                          )}
                        </div>
                      ) : lifecycleView === "trash" ? (
                        <div className="crud-row-actions">
                          {canRestore && (
                            <button
                              type="button"
                              className="btn-sm btn-primary"
                              disabled={busyKey === `${row.id}:restore`}
                              onClick={() => {
                                void restoreRow(row);
                              }}
                            >
                              {busyKey === `${row.id}:restore` ? "..." : str.actionRestore}
                            </button>
                          )}
                          {canPurge &&
                            (confirmDeleteId === row.id ? (
                              <div className="confirm-delete-inline" role="group" aria-label={`${str.actionPurge} ${label}`}>
                                <div className="confirm-delete-copy">
                                  <span className="confirm-delete-hint">
                                    {interpolate(str.confirmHint, { word: purgeWord })}
                                  </span>
                                </div>
                                <input
                                  type="text"
                                  className="confirm-delete-input"
                                  value={confirmDeleteText}
                                  onChange={(event) => setConfirmDeleteText(event.target.value)}
                                  placeholder={str.confirmPlaceholder}
                                  autoFocus
                                />
                                <div className="confirm-delete-actions">
                                  <button
                                    type="button"
                                    className="btn-sm btn-danger"
                                    disabled={
                                      confirmDeleteText.toLowerCase() !== purgeWord.toLowerCase() ||
                                      busyKey === `${row.id}:purge`
                                    }
                                    onClick={() => {
                                      void purgeRow(row);
                                    }}
                                  >
                                    {busyKey === `${row.id}:purge` ? "..." : str.actionConfirm}
                                  </button>
                                  <button type="button" className="btn-sm btn-secondary" onClick={cancelPurge}>
                                    {str.actionCancel}
                                  </button>
                                </div>
                              </div>
                            ) : (
                              <button
                                type="button"
                                className="btn-sm btn-danger"
                                disabled={busyKey === `${row.id}:purge`}
                                onClick={() => {
                                  setConfirmDeleteId(row.id);
                                  setConfirmDeleteText("");
                                }}
                              >
                                {str.actionPurge}
                              </button>
                            ))}
                        </div>
                      ) : (
                        <div className="crud-row-actions">
                          {canEdit && (
                            <button
                              type="button"
                              className="btn-sm btn-secondary"
                              onClick={() => (onExternalEdit ? onExternalEdit(row) : openEdit(row))}
                            >
                              {str.actionEdit}
                            </button>
                          )}
                          {visibleRowActions.map((action) => (
                            <button
                              key={action.id}
                              type="button"
                              className={buttonClass(action.kind)}
                              disabled={busyKey === `${row.id}:${action.id}`}
                              onClick={() => {
                                void runRowAction(action, row);
                              }}
                            >
                              {busyKey === `${row.id}:${action.id}` ? "..." : formatFieldText(action.label)}
                            </button>
                          ))}
                          {canArchive && (
                            <button
                              type="button"
                              className="btn-sm btn-secondary"
                              disabled={busyKey === `${row.id}:archive`}
                              onClick={() => {
                                void archiveRow(row);
                              }}
                            >
                              {busyKey === `${row.id}:archive` ? "..." : str.actionArchive}
                            </button>
                          )}
                          {canTrash && (
                            <button
                              type="button"
                              className="btn-sm btn-danger"
                              disabled={busyKey === `${row.id}:trash`}
                              onClick={() => {
                                void trashRow(row);
                              }}
                            >
                              {busyKey === `${row.id}:trash` ? "..." : str.actionTrash}
                            </button>
                          )}
                        </div>
                      )}
                    </td>
                    ) : null}
                  </tr>
                );
              })}
            </tbody>
          </table>
          {hasMore && featureFlags?.pagination !== false && (
            <div className="crud-load-more">
              <button
                type="button"
                className="btn-secondary"
                disabled={loadingMore}
                onClick={() => { void loadMore(); }}
              >
                {loadingMore ? str.statusLoading : str.loadMore}
              </button>
            </div>
          )}
        </div>
      )}
    </CrudPageShell>
  );
}
