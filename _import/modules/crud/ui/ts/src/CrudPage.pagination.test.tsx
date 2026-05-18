// @vitest-environment jsdom

import type { ReactNode } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CrudPage } from "./CrudPage";

type Row = { id: string; name: string };

vi.mock("@devpablocristo/core-browser/crud", () => ({
  CrudPageShell: ({
    search,
    children,
  }: {
    search?: { value: string; onChange: (value: string) => void; placeholder?: string };
    children?: ReactNode;
  }) => (
    <section>
      {search ? (
        <input
          aria-label="Search"
          value={search.value}
          onChange={(event) => search.onChange(event.target.value)}
          placeholder={search.placeholder}
        />
      ) : null}
      {children}
    </section>
  ),
  parsePaginatedResponse: (data: { items?: unknown[]; hasMore?: boolean; nextCursor?: string } | unknown[]) =>
    Array.isArray(data)
      ? { items: data, hasMore: false }
      : {
          items: data.items ?? [],
          hasMore: Boolean(data.hasMore),
          nextCursor: data.nextCursor,
        },
}));

const baseProps = {
  basePath: "/v1/widgets",
  label: "widget",
  labelPlural: "widgets",
  labelPluralCap: "Widgets",
  columns: [{ key: "name" as const, header: "Name" }],
  formFields: [],
  searchText: (row: Row) => row.name,
  toFormValues: () => ({}),
  isValid: () => true,
};

describe("CrudPage pagination", () => {
  it("uses cursor as the default pagination parameter", async () => {
    const json = vi
      .fn()
      .mockResolvedValueOnce({ items: [{ id: "1", name: "One" }], hasMore: true, nextCursor: "abc" })
      .mockResolvedValueOnce({ items: [{ id: "2", name: "Two" }], hasMore: false });

    render(<CrudPage<Row> {...baseProps} httpClient={{ json }} />);

    expect(await screen.findByText("One")).not.toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));

    await waitFor(() => expect(json).toHaveBeenLastCalledWith("/v1/widgets?limit=100&cursor=abc"));
    expect(await screen.findByText("Two")).not.toBeNull();
  });

  it("allows after as a legacy pagination parameter", async () => {
    const json = vi
      .fn()
      .mockResolvedValueOnce({ items: [{ id: "1", name: "One" }], hasMore: true, nextCursor: "abc" })
      .mockResolvedValueOnce({ items: [], hasMore: false });

    render(<CrudPage<Row> {...baseProps} paginationCursorParam="after" httpClient={{ json }} />);

    await screen.findByText("One");
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));

    await waitFor(() => expect(json).toHaveBeenLastCalledWith("/v1/widgets?limit=100&after=abc"));
  });
});
