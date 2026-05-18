// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it, beforeEach } from "vitest";
import { CrudUiPreferencesPanel } from "../src/CrudUiPreferencesPanel";

describe("CrudUiPreferencesPanel", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("permite limitar switches visibles y ocultar vista por defecto", async () => {
    render(
      <CrudUiPreferencesPanel
        storageKey="test.crud-ui"
        resources={[{ resourceId: "customers", label: "Clientes" }]}
        loadPageConfig={async () => ({
          viewModes: [{ id: "list", label: "Lista", path: "list", isDefault: true }],
        })}
        hideResourceCardHeader
        hideDefaultViewSelector
        featureKeys={[
          ["creatorFilter", "Filtro de responsable"],
          ["headerQuickFilterStrip", "Filtros rápidos en cabecera"],
          ["csvToolbar", "Acciones CSV"],
        ]}
      />,
    );

    expect(await screen.findByText("Filtro de responsable")).toBeTruthy();
    expect(screen.getByText("Filtros rápidos en cabecera")).toBeTruthy();
    expect(screen.getByText("Acciones CSV")).toBeTruthy();
    expect(screen.queryByText("Paginación")).toBeNull();
    expect(screen.queryByText("Columna tags")).toBeNull();
    expect(screen.queryByRole("combobox")).toBeNull();
  });
});
