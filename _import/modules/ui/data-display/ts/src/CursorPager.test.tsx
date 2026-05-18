// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CursorPager } from "./CursorPager";

describe("CursorPager", () => {
  it("calls onLoadMore when more results are available", () => {
    const onLoadMore = vi.fn();
    render(<CursorPager hasMore onLoadMore={onLoadMore} />);

    fireEvent.click(screen.getByRole("button", { name: "Cargar más" }));

    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });

  it("disables the button while loading", () => {
    render(<CursorPager hasMore loading onLoadMore={() => undefined} />);

    const button = screen.getByRole("button", { name: "Cargando..." });
    expect((button as HTMLButtonElement).disabled).toBe(true);
    expect(button.getAttribute("aria-busy")).toBe("true");
  });

  it("announces the end state when there are no more results", () => {
    render(<CursorPager hasMore={false} onLoadMore={() => undefined} />);

    expect(screen.getByText("No hay más resultados")).not.toBeNull();
    expect(screen.queryByRole("button")).toBeNull();
  });
});
