import { describe, expect, it } from "vitest";
import { compareUnknown } from "../src/columnSort";

describe("compareUnknown", () => {
  it("ordena números asc y desc", () => {
    expect(compareUnknown(2, 10, "asc")).toBeLessThan(0);
    expect(compareUnknown(2, 10, "desc")).toBeGreaterThan(0);
  });

  it("ordena texto con locale numérico", () => {
    expect(compareUnknown("item-2", "item-10", "asc")).toBeLessThan(0);
  });

  it("ordena fechas ISO como string", () => {
    expect(compareUnknown("2026-04-01T00:00:00Z", "2026-04-02T00:00:00Z", "asc")).toBeLessThan(0);
  });
});
