import { describe, expect, it } from "vitest";
import { crudItemPath, crudListPath } from "../src/restPaths";

describe("crudListPath", () => {
  it("appends archived segment when requested", () => {
    expect(crudListPath("/v1/widgets", true)).toBe("/v1/widgets/archived");
  });

  it("trims trailing slash on base", () => {
    expect(crudListPath("/v1/widgets/", false)).toBe("/v1/widgets");
  });
});

describe("crudItemPath", () => {
  it("builds restore and hard paths", () => {
    expect(crudItemPath("/v1/w", "id-1", "restore")).toBe("/v1/w/id-1/restore");
    expect(crudItemPath("/v1/w", "id-1", "hard")).toBe("/v1/w/id-1/hard");
  });
});
