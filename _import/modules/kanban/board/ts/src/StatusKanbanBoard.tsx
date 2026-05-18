/**
 * Tablero Kanban genérico por estado (columnas = ids de estado).
 * El padre posee `items`, reacciona a `onMoveCard` (optimistic + API) y a `onCardOpen`.
 */
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import { CrudPageShell } from "@devpablocristo/core-browser/crud";
import React, { useCallback, useEffect, useMemo, useRef, useState, type ReactElement, type ReactNode, type RefObject } from "react";
import { kanbanCollisionDetection } from "./collision";
import "./StatusKanbanBoard.css";

export type KanbanColumnDef = { id: string; label: string };

/** Evita abrir detalle justo después de soltar drag. */
export type SuppressCardOpen = { id: string | null; until: number };

export type StatusKanbanBoardProps<T extends { id: string }> = {
  columns: KanbanColumnDef[];
  columnIdSet: Set<string>;
  getRowColumnId: (row: T) => string;
  fallbackColumnId: string;
  items: T[];
  loading: boolean;
  error: string | null;
  onReload?: () => void | Promise<void>;
  /**
   * Mover tarjeta a otra columna o reordenar dentro de la misma.
   * overItemId: id de la tarjeta sobre/antes de la que se soltó.
   */
  onMoveCard: (id: string, targetColumnId: string, overItemId?: string) => void;
  resolveDropColumnId: (overId: string | undefined, items: T[]) => string | null;
  sortInColumn?: (a: T, b: T) => number;
  filterRow?: (row: T, queryLower: string) => boolean;
  renderCard: (args: {
    row: T;
    onOpen: () => void;
    suppressOpenRef: RefObject<SuppressCardOpen>;
  }) => ReactElement;
  renderOverlayCard: (row: T) => ReactElement;
  onCardOpen: (row: T) => void;
  title: string;
  subtitle?: string;
  headerLeadSlot?: ReactNode;
  searchPlaceholder?: string;
  statsLine: (visibleCount: number, totalCount: number) => string;
  emptyState?: ReactNode;
  columnFooter?: (columnId: string) => ReactNode;
  isRowDraggable?: (row: T) => boolean;
  isColumnDroppable?: (columnId: string) => boolean;
  searchInputClassName?: string;
  afterStats?: ReactNode;
  toolbarButtonRow?: ReactNode;
  externalSearch?: string;
};

/* ─── Columna ─── */

function ColumnBody({
  columnId,
  label,
  count,
  boardDragging,
  droppable,
  highlighted,
  registerBodyRef,
  children,
  footer,
  showPlaceholder,
  placeholderHeight,
}: {
  columnId: string;
  label: string;
  count: number;
  boardDragging: boolean;
  droppable: boolean;
  highlighted?: boolean;
  registerBodyRef?: (columnId: string, element: HTMLDivElement | null) => void;
  children: ReactNode;
  footer?: ReactNode;
  showPlaceholder?: boolean;
  placeholderHeight?: number;
}) {
  const droppableId = `col-${columnId}`;
  const { setNodeRef, isOver } = useDroppable({ id: droppableId, disabled: !droppable });
  const setRefs = (element: HTMLDivElement | null) => {
    setNodeRef(element);
    registerBodyRef?.(columnId, element);
  };
  return (
    <div
      ref={setRefs}
      className={`m-kanban__column-body ${!droppable ? "m-kanban__column-body--no-drop" : ""} ${isOver || highlighted ? "m-kanban__column-body--over" : ""} ${boardDragging ? "m-kanban__column-body--dragging" : ""}`}
      data-column={columnId}
    >
      <div className="m-kanban__column-head">
        <span className="m-kanban__column-label">{label}</span>
        <div className="m-kanban__column-head-actions">
          <span className="m-kanban__column-count">{count}</span>
          <span className="m-kanban__column-menu" aria-hidden="true">
            ···
          </span>
        </div>
      </div>
      <div className="m-kanban__column-scroll">
        {children}
        {showPlaceholder ? (
          <div className="m-kanban__drop-placeholder" style={{ height: placeholderHeight || 80 }} />
        ) : null}
      </div>
      {footer}
    </div>
  );
}

/* ─── Tarjeta ─── */

function DraggableCard<T extends { id: string }>({
  row,
  renderCard,
  onCardOpen,
  suppressOpenRef,
  draggable,
  onHeightCapture,
  cardRefsMap,
}: {
  row: T;
  renderCard: StatusKanbanBoardProps<T>["renderCard"];
  onCardOpen: (row: T) => void;
  suppressOpenRef: RefObject<SuppressCardOpen>;
  draggable: boolean;
  onHeightCapture: (id: string, h: number) => void;
  cardRefsMap: React.MutableRefObject<Map<string, HTMLDivElement>>;
}) {
  const { attributes, listeners, setNodeRef: setDragRef, transform, isDragging } = useDraggable({
    id: row.id,
    disabled: !draggable,
  });
  const { setNodeRef: setDropRef } = useDroppable({ id: row.id });
  const localRef = useRef<HTMLDivElement>(null);
  const measuredRef = useRef(false);

  useEffect(() => {
    if (localRef.current && !measuredRef.current) {
      measuredRef.current = true;
      onHeightCapture(row.id, localRef.current.getBoundingClientRect().height);
    }
  });

  const style: React.CSSProperties = {
    transform: CSS.Translate.toString(transform),
    opacity: isDragging ? 0 : 1,
    height: isDragging ? 0 : undefined,
    overflow: isDragging ? "hidden" as const : undefined,
    margin: isDragging ? 0 : undefined,
    padding: isDragging ? 0 : undefined,
  };
  const handleOpen = () => {
    const s = suppressOpenRef.current;
    if (s != null && s.id === row.id && Date.now() < s.until) return;
    onCardOpen(row);
  };
  const setRefs = (el: HTMLDivElement | null) => {
    setDragRef(el);
    setDropRef(el);
    (localRef as React.MutableRefObject<HTMLDivElement | null>).current = el;
    if (el) {
      cardRefsMap.current.set(row.id, el);
    } else {
      cardRefsMap.current.delete(row.id);
    }
  };
  return (
    <div ref={setRefs} style={style} {...listeners} {...attributes} draggable={false}>
      {renderCard({ row, onOpen: handleOpen, suppressOpenRef })}
    </div>
  );
}

function chooseDropColumnId<T extends { id: string }>(options: {
  event: DragEndEvent;
  items: T[];
  currentColumnId: string;
  columnIdSet: Set<string>;
  resolveDropColumnId: (overId: string | undefined, items: T[]) => string | null;
}) {
  const { event, items, currentColumnId, columnIdSet, resolveDropColumnId } = options;
  const directOverId = event.over?.id != null ? String(event.over.id) : undefined;

  let direct: string | null = null;
  if (directOverId?.startsWith("col-")) {
    const colId = directOverId.slice(4);
    direct = columnIdSet.has(colId) ? colId : null;
  } else {
    direct = resolveDropColumnId(directOverId, items);
  }

  if (direct && direct !== currentColumnId) return direct;

  const collisionCandidates =
    event.collisions
      ?.map((collision) => {
        const collisionId = String(collision.id);
        if (collisionId.startsWith("col-")) {
          const colId = collisionId.slice(4);
          return columnIdSet.has(colId) ? colId : null;
        }
        return resolveDropColumnId(collisionId, items);
      })
      .filter((columnId): columnId is string => Boolean(columnId)) ?? [];

  return collisionCandidates.find((columnId) => columnId !== currentColumnId) ?? direct;
}

function chooseHoverColumnId<T extends { id: string }>(options: {
  event: DragOverEvent;
  items: T[];
  columnIdSet: Set<string>;
  resolveDropColumnId: (overId: string | undefined, items: T[]) => string | null;
  pointer: { x: number; y: number };
  columnBodyRefs: Map<string, HTMLDivElement>;
}) {
  const { event, items, columnIdSet, resolveDropColumnId, pointer, columnBodyRefs } = options;
  for (const [columnId, element] of columnBodyRefs.entries()) {
    const rect = element.getBoundingClientRect();
    if (rect.width === 0 && rect.height === 0) continue;
    if (pointer.x >= rect.left && pointer.x <= rect.right && pointer.y >= rect.top && pointer.y <= rect.bottom) {
      return columnId;
    }
  }

  const directOverId = event.over?.id != null ? String(event.over.id) : undefined;

  if (directOverId?.startsWith("col-")) {
    const colId = directOverId.slice(4);
    if (columnIdSet.has(colId)) return colId;
  }

  const direct = resolveDropColumnId(directOverId, items);
  if (direct) return direct;

  const collisionCandidates =
    event.collisions
      ?.map((collision) => {
        const collisionId = String(collision.id);
        if (collisionId.startsWith("col-")) {
          const colId = collisionId.slice(4);
          return columnIdSet.has(colId) ? colId : null;
        }
        return resolveDropColumnId(collisionId, items);
      })
      .filter((columnId): columnId is string => Boolean(columnId)) ?? [];

  return collisionCandidates[0] ?? null;
}

/* ─── Board ─── */

export function StatusKanbanBoard<T extends { id: string }>(props: StatusKanbanBoardProps<T>): ReactElement {
  const {
    columns,
    columnIdSet,
    getRowColumnId,
    fallbackColumnId,
    items,
    loading,
    error,
    onReload,
    onMoveCard,
    resolveDropColumnId,
    sortInColumn,
    filterRow,
    renderCard,
    renderOverlayCard,
    onCardOpen,
    title,
    subtitle,
    headerLeadSlot,
    searchPlaceholder,
    statsLine,
    emptyState,
    columnFooter,
    isRowDraggable,
    isColumnDroppable,
    searchInputClassName,
    afterStats,
    toolbarButtonRow,
    externalSearch,
  } = props;

  const rowDraggable = isRowDraggable ?? (() => true);
  const columnDroppable = isColumnDroppable ?? (() => true);

  const [internalSearch, setInternalSearch] = useState("");
  const search = externalSearch ?? internalSearch;
  const [activeDrag, setActiveDrag] = useState<T | null>(null);
  const [dragHeight, setDragHeight] = useState(0);
  const [overColId, setOverColId] = useState<string | null>(null);
  const cardHeightsRef = useRef<Map<string, number>>(new Map());
  const cardRefsMap = useRef<Map<string, HTMLDivElement>>(new Map());
  const columnBodyRefs = useRef<Map<string, HTMLDivElement>>(new Map());
  const itemsRef = useRef<T[]>([]);
  itemsRef.current = items;
  const suppressCardOpenRef = useRef<SuppressCardOpen>({ id: null, until: 0 });
  const activeDragIdRef = useRef<string | null>(null);
  const lastPointerX = useRef(0);
  const lastPointerY = useRef(0);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
  );

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return items;
    return items.filter((row) => (filterRow ? filterRow(row, q) : true));
  }, [items, search, filterRow]);

  const byColumn = useMemo(() => {
    const map = new Map<string, T[]>();
    for (const col of columns) {
      map.set(col.id, []);
    }
    for (const row of filtered) {
      let c = getRowColumnId(row);
      if (!columnIdSet.has(c)) {
        c = fallbackColumnId;
      }
      const bucket = map.get(c);
      if (bucket) {
        bucket.push(row);
      } else {
        const fb = map.get(fallbackColumnId);
        if (fb) fb.push(row);
      }
    }
    if (sortInColumn) {
      for (const [, list] of map) {
        list.sort(sortInColumn);
      }
    }
    return map;
  }, [filtered, columns, columnIdSet, getRowColumnId, fallbackColumnId, sortInColumn]);

  const boardDragging = activeDrag != null;

  /**
   * Dado un targetColumnId y la posición Y del pointer, encontrar la tarjeta
   * más cercana en esa columna y devolver su id. Usa rects DOM reales.
   */
  const findOverItemInColumn = useCallback((targetColId: string, pointerY: number, activeId: string): string | undefined => {
    const colItems = byColumn.get(targetColId);
    if (!colItems || colItems.length === 0) return undefined;

    let closestId: string | undefined;
    let closestDist = Infinity;

    for (const item of colItems) {
      if (item.id === activeId) continue;
      const el = cardRefsMap.current.get(item.id);
      if (!el) continue;
      const rect = el.getBoundingClientRect();
      if (rect.height === 0) continue; // collapsed
      const centerY = rect.top + rect.height / 2;
      const dist = Math.abs(pointerY - centerY);
      if (dist < closestDist) {
        closestDist = dist;
        closestId = item.id;
      }
    }
    return closestId;
  }, [byColumn]);

  const handleDragStart = (e: DragStartEvent) => {
    const id = String(e.active.id);
    activeDragIdRef.current = id;
    setActiveDrag(itemsRef.current.find((x) => x.id === id) ?? null);
    setDragHeight(cardHeightsRef.current.get(id) ?? e.active.rect.current.initial?.height ?? 80);
  };

  const handleHeightCapture = useCallback((id: string, h: number) => {
    cardHeightsRef.current.set(id, h);
  }, []);

  const handleDragOver = (e: DragOverEvent) => {
    setOverColId(
      chooseHoverColumnId({
        event: e,
        items: itemsRef.current,
        columnIdSet,
        resolveDropColumnId,
        pointer: { x: lastPointerX.current, y: lastPointerY.current },
        columnBodyRefs: columnBodyRefs.current,
      }),
    );
  };

  // Trackear posición del pointer durante el drag
  useEffect(() => {
    if (!boardDragging) return;
    const handler = (e: PointerEvent) => {
      lastPointerX.current = e.clientX;
      lastPointerY.current = e.clientY;
    };
    window.addEventListener("pointermove", handler);
    return () => window.removeEventListener("pointermove", handler);
  }, [boardDragging]);

  const registerColumnBodyRef = useCallback((columnId: string, element: HTMLDivElement | null) => {
    if (element) {
      columnBodyRefs.current.set(columnId, element);
      return;
    }
    columnBodyRefs.current.delete(columnId);
  }, []);

  const handleDragEnd = (e: DragEndEvent) => {
    suppressCardOpenRef.current = { id: String(e.active.id), until: Date.now() + 260 };
    activeDragIdRef.current = null;
    const { active, over } = e;
    setActiveDrag(null);
    setOverColId(null);
    setDragHeight(0);

    const id = String(active.id);
    const snapshot = itemsRef.current;
    const row = snapshot.find((x) => x.id === id);
    if (!row) return;
    if (!over) return;

    const currentColumnId = getRowColumnId(row);
    const targetCol = chooseDropColumnId({
      event: e,
      items: snapshot,
      currentColumnId,
      columnIdSet,
      resolveDropColumnId,
    });
    if (!targetCol) return;

    // Determinar overItemId: la tarjeta más cercana al pointer en la columna destino
    const overItemId = findOverItemInColumn(targetCol, lastPointerY.current, id);
    const sameColumn = currentColumnId === targetCol;
    if (sameColumn && !overItemId) return;
    if (sameColumn && overItemId === id) return;

    if (overItemId !== undefined) {
      onMoveCard(id, targetCol, overItemId);
      return;
    }
    onMoveCard(id, targetCol);
  };

  const handleDragCancel = () => {
    const id = activeDragIdRef.current;
    if (id) suppressCardOpenRef.current = { id, until: Date.now() + 260 };
    activeDragIdRef.current = null;
    setActiveDrag(null);
    setOverColId(null);
    setDragHeight(0);
  };

  const totalVisible = filtered.length;

  return (
    <div className="m-kanban">
      <CrudPageShell
        title={title}
        subtitle={subtitle}
        headerLeadSlot={
          <>
            {headerLeadSlot != null ? <div className="crud-list-header-lead">{headerLeadSlot}</div> : null}
            <p className="text-secondary" aria-live="polite">
              {loading ? "Cargando…" : statsLine(totalVisible, items.length)}
            </p>
            {afterStats ? <div className="crud-list-header-lead">{afterStats}</div> : null}
          </>
        }
        search={externalSearch == null ? {
          value: internalSearch,
          onChange: setInternalSearch,
          placeholder: searchPlaceholder ?? "Buscar...",
          ariaLabel: searchPlaceholder ?? "Buscar...",
          inputClassName: [searchInputClassName, "m-kanban__search"].filter(Boolean).join(" ").trim(),
        } : undefined}
        headerActions={onReload != null || toolbarButtonRow != null ? (
          <>
            {onReload != null ? (
              <button
                type="button"
                className="btn-sm btn-secondary"
                onClick={() => {
                  void onReload();
                }}
              >
                Reload
              </button>
            ) : null}
            {toolbarButtonRow}
          </>
        ) : undefined}
        error={error ? (
          <p className="m-kanban__error" role="alert">
            {error}
          </p>
        ) : undefined}
      >
        <>
          {!loading && items.length === 0 && emptyState ? <div className="m-kanban__empty">{emptyState}</div> : null}

          <DndContext
            sensors={sensors}
            collisionDetection={kanbanCollisionDetection}
            autoScroll={false}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
            onDragCancel={handleDragCancel}
          >
            <div className="m-kanban__board">
              {columns.map((col) => {
                const columnItems = byColumn.get(col.id) ?? [];
                return (
                  <div key={col.id} className="m-kanban__column-shell">
                    <ColumnBody
                      columnId={col.id}
                      label={col.label}
                      count={columnItems.length}
                      boardDragging={boardDragging}
                      droppable={columnDroppable(col.id)}
                      highlighted={overColId === col.id}
                      registerBodyRef={registerColumnBodyRef}
                      footer={columnFooter?.(col.id)}
                      showPlaceholder={overColId === col.id && activeDrag != null}
                      placeholderHeight={dragHeight}
                    >
                      {columnItems.map((row) => (
                        <DraggableCard
                          key={row.id}
                          row={row}
                          renderCard={renderCard}
                          onCardOpen={onCardOpen}
                          suppressOpenRef={suppressCardOpenRef}
                          draggable={rowDraggable(row)}
                          onHeightCapture={handleHeightCapture}
                          cardRefsMap={cardRefsMap}
                        />
                      ))}
                    </ColumnBody>
                  </div>
                );
              })}
            </div>
            <DragOverlay dropAnimation={{ duration: 160, easing: "cubic-bezier(0.25, 1, 0.5, 1)" }}>
              {activeDrag ? renderOverlayCard(activeDrag) : null}
            </DragOverlay>
          </DndContext>
        </>
      </CrudPageShell>
    </div>
  );
}
