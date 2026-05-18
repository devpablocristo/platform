// @vitest-environment jsdom

import type { ComponentProps, ReactNode, RefObject } from "react";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StatusKanbanBoard, type SuppressCardOpen } from "../src/StatusKanbanBoard";

type CardRow = {
  id: string;
  title: string;
  status: string;
};

const dndState = vi.hoisted(() => ({
  lastProps: null as null | Record<string, unknown>,
}));

vi.mock("@dnd-kit/utilities", () => ({
  CSS: {
    Translate: {
      toString: () => "",
    },
  },
}));

vi.mock("@dnd-kit/core", () => ({
  DndContext: ({ children, ...props }: { children: ReactNode }) => {
    dndState.lastProps = props as Record<string, unknown>;
    return <div data-testid="dnd-context">{children}</div>;
  },
  DragOverlay: ({ children }: { children?: ReactNode }) => <div data-testid="drag-overlay">{children}</div>,
  PointerSensor: function PointerSensor() {},
  useSensor: () => ({ sensor: "pointer" }),
  useSensors: (...sensors: unknown[]) => sensors,
  useDroppable: () => ({
    setNodeRef: () => undefined,
    isOver: false,
  }),
  useDraggable: () => ({
    setNodeRef: () => undefined,
    transform: null,
    isDragging: false,
    attributes: {},
    listeners: {},
  }),
}));

vi.mock("@devpablocristo/core-browser/crud", () => ({
  CrudPageShell: ({
    title,
    subtitle,
    headerLeadSlot,
    search,
    headerActions,
    error,
    children,
  }: {
    title: ReactNode;
    subtitle?: ReactNode;
    headerLeadSlot?: ReactNode;
    search?: { value: string; onChange: (value: string) => void; placeholder?: string; ariaLabel?: string };
    headerActions?: ReactNode;
    error?: ReactNode;
    children?: ReactNode;
  }) => (
    <section>
      <h1>{title}</h1>
      {subtitle ? <p>{subtitle}</p> : null}
      {headerLeadSlot}
      {search ? (
        <input
          aria-label={search.ariaLabel ?? "Search"}
          placeholder={search.placeholder}
          value={search.value}
          onChange={(event) => search.onChange(event.target.value)}
        />
      ) : null}
      <div data-testid="header-actions">{headerActions}</div>
      {error}
      {children}
    </section>
  ),
}));

const columns = [
  { id: "todo", label: "To do" },
  { id: "done", label: "Done" },
];

function renderBoard(overrides?: Partial<ComponentProps<typeof StatusKanbanBoard<CardRow>>>) {
  const onReload = vi.fn();
  const onMoveCard = vi.fn();
  const onCardOpen = vi.fn();
  const props: ComponentProps<typeof StatusKanbanBoard<CardRow>> = {
    columns,
    columnIdSet: new Set(columns.map((column) => column.id)),
    getRowColumnId: (row) => row.status,
    fallbackColumnId: "todo",
    items: [
      { id: "1", title: "Ada", status: "todo" },
      { id: "2", title: "Grace", status: "unknown" },
    ],
    loading: false,
    error: null,
    onReload,
    onMoveCard,
    resolveDropColumnId: (overId) => (overId === "col-done" ? "done" : null),
    filterRow: (row, queryLower) => row.title.toLowerCase().includes(queryLower),
    renderCard: ({ row, onOpen }: { row: CardRow; onOpen: () => void; suppressOpenRef: RefObject<SuppressCardOpen> }) => (
      <button type="button" onClick={onOpen}>
        Open {row.title}
      </button>
    ),
    renderOverlayCard: (row) => <div>Overlay {row.title}</div>,
    onCardOpen,
    title: "Work orders",
    subtitle: "Board",
    searchPlaceholder: "Search cards",
    statsLine: (visible, total) => `${visible} visible / ${total} total`,
    toolbarButtonRow: <button type="button">Bulk action</button>,
    ...overrides,
  };

  render(<StatusKanbanBoard<CardRow> {...props} />);
  return { onReload, onMoveCard, onCardOpen };
}

describe("StatusKanbanBoard", () => {
  it("renders search, reload, toolbar actions, and filters cards", async () => {
    const { onReload, onCardOpen } = renderBoard();

    expect(screen.getByText("Work orders")).toBeTruthy();
    expect(screen.getByText("2 visible / 2 total")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Reload" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Bulk action" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    expect(onReload).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByLabelText("Search cards"), { target: { value: "grace" } });
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Open Ada" })).toBeNull();
    });
    expect(screen.getByRole("button", { name: "Open Grace" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Open Grace" }));
    expect(onCardOpen).toHaveBeenCalledWith({ id: "2", title: "Grace", status: "unknown" });
  });

  it("moves a card when drag ends in a different resolved column", async () => {
    const { onMoveCard } = renderBoard({
      items: [{ id: "1", title: "Ada", status: "todo" }],
    });

    const props = dndState.lastProps as {
      onDragStart: (event: { active: { id: string } }) => void;
      onDragEnd: (event: { active: { id: string }; over: { id: string } }) => void;
    };

    await act(async () => {
      props.onDragStart({ active: { id: "1" } });
    });
    expect(screen.getByText("Overlay Ada")).toBeTruthy();

    await act(async () => {
      props.onDragEnd({ active: { id: "1" }, over: { id: "col-done" } });
    });

    await waitFor(() => {
      expect(onMoveCard).toHaveBeenCalledWith("1", "done");
    });
  });

  it("highlights the resolved target column while dragging over an existing card", async () => {
    renderBoard({
      items: [
        { id: "1", title: "Ada", status: "todo" },
        { id: "2", title: "Grace", status: "done" },
      ],
      resolveDropColumnId: (overId) => (overId === "2" ? "done" : null),
    });

    const props = dndState.lastProps as {
      onDragStart: (event: { active: { id: string } }) => void;
      onDragOver: (event: { over: { id: string }; collisions?: Array<{ id: string }> }) => void;
    };

    await act(async () => {
      props.onDragStart({ active: { id: "1" } });
    });

    const draggingProps = dndState.lastProps as {
      onDragOver: (event: { over: { id: string }; collisions?: Array<{ id: string }> }) => void;
    };

    await act(async () => {
      draggingProps.onDragOver({ over: { id: "1" }, collisions: [{ id: "2" }] });
    });

    expect(screen.getByText("Done").closest(".m-kanban__column-body")?.className).toContain("m-kanban__column-body--over");
  });
});
