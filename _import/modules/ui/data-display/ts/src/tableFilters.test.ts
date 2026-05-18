import { describe, expect, it } from "vitest";

import { applyTableFilters, getTableFilterOptions } from "./tableFilters";

type Row = {
  name: string;
  category: string;
  contractor: string;
  price: string;
};

const rows: Row[] = [
  { name: "Siembra", category: "Implantacion", contractor: "Acme", price: "100" },
  { name: "Pulverizacion", category: "Proteccion", contractor: "Acme", price: "80" },
  { name: "Cosecha", category: "Cosecha", contractor: "Campo Sur", price: "120" },
  { name: "Fertilizacion", category: "Proteccion", contractor: "Campo Sur", price: "80" },
];

describe("tableFilters", () => {
  it("filters select arrays by exact normalized value", () => {
    expect(
      applyTableFilters(rows, { contractor: ["acme"] }).map((row) => row.name)
    ).toEqual(["Siembra", "Pulverizacion"]);
  });

  it("filters scalar values by partial text", () => {
    expect(
      applyTableFilters(rows, { name: "cion" }).map((row) => row.name)
    ).toEqual(["Pulverizacion", "Fertilizacion"]);
  });

  it("builds options from rows matching the other active filters", () => {
    expect(getTableFilterOptions(rows, { category: ["Proteccion"] }, "contractor")).toEqual([
      "Acme",
      "Campo Sur",
    ]);

    expect(getTableFilterOptions(rows, { contractor: ["Acme"] }, "category")).toEqual([
      "Implantacion",
      "Proteccion",
    ]);
  });
});
