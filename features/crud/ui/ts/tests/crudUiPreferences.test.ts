// @vitest-environment jsdom

import { describe, expect, it, beforeEach } from "vitest";
import { createCrudUiPreferencesApi } from "../src/crudUiPreferences";
import type { CrudPageConfig } from "../src/types";

describe("createCrudUiPreferencesApi", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("applies enabled view modes and default", () => {
    const api = createCrudUiPreferencesApi({
      storageKey: "t.crud-ui",
      knownResourceIds: ["products"],
    });
    api.writeState({
      products: { enabledViewModeIds: ["list"], defaultViewModeId: "list" },
    });
    const config: CrudPageConfig<{ id: string }> = {
      label: "x",
      labelPlural: "xs",
      labelPluralCap: "Xs",
      columns: [],
      formFields: [],
      searchText: () => "",
      toFormValues: () => ({}),
      isValid: () => true,
      viewModes: [
        { id: "gallery", label: "G", path: "g", isDefault: true },
        { id: "list", label: "L", path: "l" },
      ],
    };
    const next = api.applyCrudUiOverride("products", config);
    expect(next.viewModes?.map((m) => m.id)).toEqual(["list"]);
    expect(next.viewModes?.every((m) => m.isDefault === (m.id === "list"))).toBe(true);
  });

  it("merge writeState: actualizar un recurso no borra otros en localStorage", () => {
    const key = "t.crud-ui-merge";
    localStorage.setItem(
      key,
      JSON.stringify({
        products: { featureFlags: { pagination: false } },
        stock: { featureFlags: { csvToolbar: false } },
      }),
    );
    const apiStockOnly = createCrudUiPreferencesApi({
      storageKey: key,
      knownResourceIds: ["stock"],
    });
    apiStockOnly.writeState({
      stock: { featureFlags: { csvToolbar: true, pagination: true } },
    });
    const raw = JSON.parse(localStorage.getItem(key) ?? "{}") as Record<string, unknown>;
    expect(raw.products).toEqual({ featureFlags: { pagination: false } });
    expect((raw.stock as { featureFlags?: { csvToolbar?: boolean } }).featureFlags?.csvToolbar).toBe(true);
  });

  it("merges feature flags", () => {
    const api = createCrudUiPreferencesApi({
      storageKey: "t.crud-ui-2",
      knownResourceIds: ["stock"],
    });
    api.writeState({
      stock: { featureFlags: { pagination: false } },
    });
    const config: CrudPageConfig<{ id: string }> = {
      label: "s",
      labelPlural: "ss",
      labelPluralCap: "Ss",
      columns: [],
      formFields: [],
      searchText: () => "",
      toFormValues: () => ({}),
      isValid: () => true,
      featureFlags: { creatorFilter: true, csvToolbar: true },
    };
    const next = api.applyCrudUiOverride("stock", config);
    expect(next.featureFlags?.pagination).toBe(false);
    expect(next.featureFlags?.creatorFilter).toBe(true);
  });

  it("translates archivedToggle and createAction flags into config props", () => {
    const api = createCrudUiPreferencesApi({
      storageKey: "t.crud-ui-3",
      knownResourceIds: ["products"],
    });
    api.writeState({
      products: { featureFlags: { archivedToggle: false, createAction: false } },
    });
    const config: CrudPageConfig<{ id: string }> = {
      label: "p",
      labelPlural: "ps",
      labelPluralCap: "Ps",
      columns: [],
      formFields: [],
      searchText: () => "",
      toFormValues: () => ({}),
      isValid: () => true,
      supportsArchived: true,
      allowCreate: true,
    };
    const next = api.applyCrudUiOverride("products", config);
    expect(next.supportsArchived).toBe(false);
    expect(next.allowCreate).toBe(false);
  });
});
