import { describe, expect, it } from "vitest";
import { mergeRecordsPreferOverride } from "../src/mergeRecords";

describe("mergeRecordsPreferOverride", () => {
  it("keeps base-only keys first, then all override keys", () => {
    const merged = mergeRecordsPreferOverride(
      { a: 1, b: 2, c: 3 },
      { b: 20, d: 4 },
    );
    expect(merged).toEqual({ a: 1, c: 3, b: 20, d: 4 });
    expect(Object.keys(merged)).toEqual(["a", "c", "b", "d"]);
  });

  it("matches empty base", () => {
    expect(mergeRecordsPreferOverride({}, { x: 1 })).toEqual({ x: 1 });
  });

  it("matches empty override", () => {
    expect(mergeRecordsPreferOverride({ x: 1 }, {})).toEqual({ x: 1 });
  });
});
