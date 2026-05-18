// @vitest-environment jsdom

import type { ComponentProps, ReactNode } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CrudPage } from "../src/CrudPage";

type UserRow = {
  id: string;
  name: string;
};

const searchMock = vi.hoisted(() => vi.fn());

vi.mock("@devpablocristo/core-browser/search", () => ({
  search: (...args: unknown[]) => searchMock(...args),
}));

vi.mock("@devpablocristo/core-browser/crud", () => ({
  CrudPageShell: ({
    title,
    subtitle,
    search,
    headerLeadSlot,
    headerActions,
    error,
    form,
    children,
  }: {
    title: string;
    subtitle?: string;
    search?: { value: string; onChange: (value: string) => void; placeholder?: string };
    headerLeadSlot?: ReactNode;
    headerActions?: ReactNode;
    error?: ReactNode;
    form?: ReactNode;
    children?: ReactNode;
  }) => (
    <section>
      <h1>{title}</h1>
      {subtitle ? <p data-testid="subtitle">{subtitle}</p> : null}
      {search ? (
        <input
          aria-label="Search"
          value={search.value}
          onChange={(event) => search.onChange(event.target.value)}
          placeholder={search.placeholder}
        />
      ) : null}
      {headerLeadSlot}
      <div data-testid="header-actions">{headerActions}</div>
      {error}
      {form}
      <div data-testid="crud-body">{children}</div>
    </section>
  ),
  parseListItemsFromResponse: (data: { items?: unknown[] } | unknown[]) =>
    Array.isArray(data) ? data : (data.items ?? []),
}));

function renderPage(overrides?: Partial<ComponentProps<typeof CrudPage<UserRow>>>) {
  const props: ComponentProps<typeof CrudPage<UserRow>> = {
    label: "user",
    labelPlural: "users",
    labelPluralCap: "Users",
    columns: [{ key: "name", header: "Name" }],
    formFields: [{ key: "name", label: "Name", required: true }],
    searchText: (row) => row.name,
    toFormValues: (row) => ({ name: row.name }),
    isValid: (values) => String(values.name ?? "").trim().length > 0,
    createLabel: "Add user",
    dataSource: {
      list: vi.fn().mockResolvedValue([]),
      create: vi.fn().mockResolvedValue(undefined),
      restore: vi.fn().mockResolvedValue(undefined),
    },
    ...overrides,
  };

  return {
    ...render(<CrudPage<UserRow> {...props} />),
    props,
  };
}

describe("CrudPage", () => {
  it("creates a new row through the data source and reloads the list", async () => {
    const list = vi.fn().mockResolvedValue([]);
    const create = vi.fn().mockResolvedValue(undefined);

    renderPage({
      dataSource: { list, create },
    });

    expect((await screen.findAllByRole("button", { name: "Add user" })).length).toBeGreaterThan(0);

    fireEvent.click(screen.getAllByRole("button", { name: "Add user" })[0]);
    fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Ada Lovelace" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(create).toHaveBeenCalledWith({ name: "Ada Lovelace" });
    });
    await waitFor(() => {
      expect(list).toHaveBeenCalledTimes(2);
    });
  });

  it("filters with the reusable search contract and shows only matched rows", async () => {
    const rows: UserRow[] = [
      { id: "1", name: "Ada Lovelace" },
      { id: "2", name: "Grace Hopper" },
    ];
    searchMock.mockReset();
    searchMock.mockReturnValue([{ item: rows[1], score: 0.95 }]);

    renderPage({
      dataSource: { list: vi.fn().mockResolvedValue(rows) },
    });

    expect(await screen.findByText("Ada Lovelace")).toBeTruthy();
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "gra" } });

    await waitFor(() => {
      expect(searchMock).toHaveBeenCalledTimes(1);
    });
    expect(screen.queryByText("Ada Lovelace")).toBeNull();
    expect(screen.getByText("Grace Hopper")).toBeTruthy();
  });

  it("starts in archived view when initialShowArchived is true", async () => {
    const list = vi.fn().mockImplementation(async ({ archived }: { archived: boolean }) =>
      archived ? [{ id: "a1", name: "Old" }] : [{ id: "1", name: "Active user" }],
    );

    renderPage({
      supportsArchived: true,
      initialShowArchived: true,
      dataSource: { list },
    });

    expect(await screen.findByText("Old")).toBeTruthy();
    expect(list).toHaveBeenNthCalledWith(1, { archived: true });
  });

  it("switches to archived mode and restores rows through the data source", async () => {
    const archivedRow = { id: "arch-1", name: "Archived user" };
    const list = vi.fn().mockImplementation(async ({ archived }: { archived: boolean }) =>
      archived ? [archivedRow] : [{ id: "1", name: "Active user" }],
    );
    const restore = vi.fn().mockResolvedValue(undefined);

    renderPage({
      supportsArchived: true,
      dataSource: { list, restore },
    });

    expect(await screen.findByText("Active user")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Show archived" }));

    expect(await screen.findByText("Archived user")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    await waitFor(() => {
      expect(restore).toHaveBeenCalledWith(archivedRow);
    });
    expect(list).toHaveBeenNthCalledWith(1, { archived: false });
    expect(list).toHaveBeenNthCalledWith(2, { archived: true });
  });

  it("invoca onRowClick al pulsar la fila y no muestra columna Acciones si no hay acciones", async () => {
    const rows: UserRow[] = [{ id: "1", name: "Ada" }];
    const onRowClick = vi.fn();

    renderPage({
      allowEdit: false,
      allowDelete: false,
      rowActions: [],
      dataSource: { list: vi.fn().mockResolvedValue(rows) },
      onRowClick,
    });

    expect(await screen.findByText("Ada")).toBeTruthy();
    expect(screen.queryByText("Actions")).toBeNull();

    const row = screen.getByText("Ada").closest("tr");
    expect(row).toBeTruthy();
    fireEvent.click(row!);

    expect(onRowClick).toHaveBeenCalledWith(rows[0]);
  });

  it("ordena filas al pulsar la cabecera de columna (asc / desc)", async () => {
    const rows: UserRow[] = [
      { id: "1", name: "Zara" },
      { id: "2", name: "Ana" },
      { id: "3", name: "Mina" },
    ];
    searchMock.mockImplementation((q, entries) => entries);

    renderPage({
      dataSource: { list: vi.fn().mockResolvedValue(rows) },
    });

    expect(await screen.findByText("Zara")).toBeTruthy();
    const sortBtn = screen.getByRole("button", { name: /Ordenar por name/i });
    fireEvent.click(sortBtn);
    const namesAsc = screen.getAllByRole("row").slice(1).map((r) => r.textContent ?? "");
    expect(namesAsc[0]).toContain("Ana");
    expect(namesAsc[1]).toContain("Mina");
    expect(namesAsc[2]).toContain("Zara");

    fireEvent.click(sortBtn);
    await waitFor(() => {
      const namesDesc = screen.getAllByRole("row").slice(1).map((r) => r.textContent ?? "");
      expect(namesDesc[0]).toContain("Zara");
      expect(namesDesc[2]).toContain("Ana");
    });
  });

  it("hides the search input when featureFlags.searchBar is false", async () => {
    renderPage({
      dataSource: { list: vi.fn().mockResolvedValue([{ id: "1", name: "Ada Lovelace" }]) },
      featureFlags: { searchBar: false },
    });

    expect(await screen.findByText("Ada Lovelace")).toBeTruthy();
    expect(screen.queryByRole("searchbox")).toBeNull();
  });

  it("hides the load-more button when featureFlags.pagination is false", async () => {
    const rows = Array.from({ length: 5 }, (_, i) => ({ id: String(i), name: `User ${i}` }));
    renderPage({
      dataSource: { list: vi.fn().mockResolvedValue(rows) },
      featureFlags: { pagination: false },
    });

    expect(await screen.findByText("User 0")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Load more/i })).toBeNull();
  });
});
