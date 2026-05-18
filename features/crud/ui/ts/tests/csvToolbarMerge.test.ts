import { describe, expect, it, vi } from "vitest";
import type { CrudPageConfig } from "../src/types";
import { buildCsvToolbarActions, mergeCsvToolbarConfig } from "../src/csvToolbarMerge";

type Row = { id: string; name: string };

const baseConfig: CrudPageConfig<Row> = {
  label: "item",
  labelPlural: "items",
  labelPluralCap: "Items",
  basePath: "/v1/items",
  columns: [{ key: "name", header: "Name" }],
  formFields: [{ key: "name", label: "Name", required: true }],
  searchText: (row) => row.name,
  toFormValues: (row) => ({ name: row.name ?? "" }),
  toBody: (values) => ({ name: String(values.name ?? "") }),
  isValid: (values) => String(values.name ?? "").trim().length > 0,
};

describe("csvToolbarMerge", () => {
  it("mergeCsvToolbarConfig prepends csv actions", () => {
    const ui = {
      confirmClientImport: vi.fn(async () => true),
      confirmServerImport: vi.fn(async () => true),
      notify: vi.fn(),
    };
    const merged = mergeCsvToolbarConfig({
      config: baseConfig,
      entity: "items",
      mode: "client",
      allowImport: false,
      ui,
      importClientRow: vi.fn(async () => {}),
    });
    expect(merged.toolbarActions?.map((a) => a.id)).toEqual(["csv-export"]);
  });

  it("buildCsvToolbarActions wires server export when port provided", async () => {
    const download = vi.fn(async () => {});
    const actions = buildCsvToolbarActions({
      config: baseConfig,
      entity: "customers",
      mode: "server",
      allowImport: false,
      serverExport: { download },
      ui: {
        confirmClientImport: vi.fn(),
        confirmServerImport: vi.fn(),
        notify: vi.fn(),
      },
      importClientRow: vi.fn(),
    });
    const exportAction = actions.find((a) => a.id === "csv-export");
    expect(exportAction).toBeDefined();
    await exportAction!.onClick({ items: [], reload: vi.fn(), setError: vi.fn() });
    expect(download).toHaveBeenCalledWith("customers");
  });
});
