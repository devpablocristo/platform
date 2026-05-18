// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { InsightSummaryCards } from "../src/InsightSummaryCards";

describe("InsightSummaryCards", () => {
  it("renders each summary card label and value", () => {
    render(
      <InsightSummaryCards
        cards={[
          { label: "Open insights", value: 4 },
          { label: "Resolved today", value: "12" },
        ]}
      />,
    );

    expect(screen.getByText("Open insights")).toBeTruthy();
    expect(screen.getByText("4")).toBeTruthy();
    expect(screen.getByText("Resolved today")).toBeTruthy();
    expect(screen.getByText("12")).toBeTruthy();
  });
});
