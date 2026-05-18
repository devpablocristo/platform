// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { InsightCardsList } from "../src/InsightCardsList";

describe("InsightCardsList", () => {
  it("renders the empty state when there are no insights", () => {
    render(
      <InsightCardsList
        title="Insights"
        items={[]}
        emptyMessage="No active insights."
      />,
    );

    expect(screen.getByText("No active insights.")).toBeTruthy();
  });

  it("shows collapsed items first and toggles the expanded view", () => {
    const onToggleExpanded = vi.fn();

    const { rerender } = render(
      <InsightCardsList
        title="Insights"
        items={[
          { id: "1", title: "A", summary: "Summary A" },
          { id: "2", title: "B", summary: "Summary B" },
          { id: "3", title: "C", summary: "Summary C", badge: "urgent", ctaLabel: "Review", impact: "+12%" },
        ]}
        emptyMessage="No active insights."
        collapsedCount={2}
        expanded={false}
        onToggleExpanded={onToggleExpanded}
      />,
    );

    expect(screen.getByText("Summary A")).toBeTruthy();
    expect(screen.getByText("Summary B")).toBeTruthy();
    expect(screen.queryByText("Summary C")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Mostrar todos" }));
    expect(onToggleExpanded).toHaveBeenCalledTimes(1);

    rerender(
      <InsightCardsList
        title="Insights"
        items={[
          { id: "1", title: "A", summary: "Summary A" },
          { id: "2", title: "B", summary: "Summary B" },
          { id: "3", title: "C", summary: "Summary C", badge: "urgent", ctaLabel: "Review", impact: "+12%" },
        ]}
        emptyMessage="No active insights."
        collapsedCount={2}
        expanded
        onToggleExpanded={onToggleExpanded}
      />,
    );

    expect(screen.getByText("Summary C")).toBeTruthy();
    expect(screen.getByText("CTA: Review")).toBeTruthy();
    expect(screen.getByText("+12%")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Mostrar menos" })).toBeTruthy();
  });
});
