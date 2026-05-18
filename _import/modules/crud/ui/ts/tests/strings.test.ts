import { describe, expect, it } from "vitest";
import { interpolate, mergeCrudStrings, defaultCrudStrings } from "../src/strings";

describe("interpolate", () => {
  it("replaces placeholders", () => {
    expect(interpolate("Hello {{name}}", { name: "World" })).toBe("Hello World");
  });

  it("uses empty string for missing keys", () => {
    expect(interpolate("{{a}}-{{b}}", { a: "1" })).toBe("1-");
  });
});

describe("mergeCrudStrings", () => {
  it("overrides base", () => {
    const merged = mergeCrudStrings(defaultCrudStrings, { actionSave: "OK" });
    expect(merged.actionSave).toBe("OK");
    expect(merged.actionCancel).toBe(defaultCrudStrings.actionCancel);
  });
});
