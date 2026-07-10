// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LifecycleActionToolbar } from "./LifecycleActionToolbar";

describe("LifecycleActionToolbar", () => {
  it("emits active lifecycle actions", () => {
    const onBulkAction = vi.fn();

    render(
      <LifecycleActionToolbar
        selectedCount={2}
        view="active"
        createOpen={false}
        busy={false}
        onCreate={vi.fn()}
        onEdit={vi.fn()}
        onClear={vi.fn()}
        onBulkAction={onBulkAction}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Archive" }));
    fireEvent.click(screen.getByRole("button", { name: "Trash" }));

    expect(onBulkAction).toHaveBeenNthCalledWith(1, "archive");
    expect(onBulkAction).toHaveBeenNthCalledWith(2, "trash");
  });

  it("disables selection actions when nothing is selected", () => {
    render(
      <LifecycleActionToolbar
        selectedCount={0}
        view="trash"
        createOpen={false}
        busy={false}
        onCreate={vi.fn()}
        onClear={vi.fn()}
        onBulkAction={vi.fn()}
      />,
    );

    expect(screen.getByRole<HTMLButtonElement>("button", { name: "Clear" }).disabled).toBe(true);
    expect(screen.getByRole<HTMLButtonElement>("button", { name: "Restore" }).disabled).toBe(true);
    expect(screen.getByRole<HTMLButtonElement>("button", { name: "Delete" }).disabled).toBe(true);
  });
});
