// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CopilotResponsePanel } from "../src/CopilotResponsePanel";

describe("CopilotResponsePanel", () => {
  it("renders the answer, json sections, related count, and custom actions", () => {
    render(
      <CopilotResponsePanel
        answer="Try moving more appointments to the morning."
        data={{ efficiency: 0.87 }}
        sources={["calendar", "sales"]}
        warnings={["Sample warning"]}
        relatedInsightsCount={2}
        relatedInsights={[
          { id: "1", title: "Morning demand", href: "/insights/1" },
          { id: "2", title: "Cancellation spike", href: "/insights/2" },
        ]}
        relatedInsightsAction={<button type="button">Open all</button>}
      />,
    );

    expect(screen.getByText("Respuesta")).toBeTruthy();
    expect(screen.getByText("Try moving more appointments to the morning.")).toBeTruthy();
    expect(screen.getByText("Datos")).toBeTruthy();
    expect(screen.getByText("Fuentes")).toBeTruthy();
    expect(screen.getByText("Advertencias")).toBeTruthy();
    expect(screen.getByText("Relacionados: 2")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Open all" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Morning demand" }).getAttribute("href")).toBe("/insights/1");
  });

  it("renders the empty related insights state and supports custom insight rendering", () => {
    const { rerender } = render(
      <CopilotResponsePanel
        answer="All clear."
        data={null}
        sources={[]}
        warnings={[]}
        relatedInsightsCount={0}
        relatedInsights={[]}
        emptyRelatedInsightsMessage="Nothing active."
      />,
    );

    expect(screen.getByText("Nothing active.")).toBeTruthy();

    rerender(
      <CopilotResponsePanel
        answer="All clear."
        data={null}
        sources={[]}
        warnings={[]}
        relatedInsightsCount={1}
        relatedInsights={[{ id: "x", title: "Custom insight" }]}
        renderRelatedInsight={(item) => <div key={item.id}>Rendered {item.title}</div>}
      />,
    );

    expect(screen.getByText("Rendered Custom insight")).toBeTruthy();
  });
});
