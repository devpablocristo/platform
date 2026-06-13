import { describe, expect, it } from "vitest";
import { crudItemPath, crudListPath } from "../src/restPaths";

describe("crudListPath", () => {
  it("builds list paths by lifecycle view", () => {
    expect(crudListPath("/v1/widgets", "active")).toBe("/v1/widgets");
    expect(crudListPath("/v1/widgets", "archived")).toBe("/v1/widgets/archived");
    expect(crudListPath("/v1/widgets", "trash")).toBe("/v1/widgets/trash");
  });

  it("trims trailing slash on base", () => {
    expect(crudListPath("/v1/widgets/", "active")).toBe("/v1/widgets");
  });
});

describe("crudItemPath", () => {
  it("builds lifecycle action paths", () => {
    expect(crudItemPath("/v1/w", "id-1", "archive")).toBe("/v1/w/id-1/archive");
    expect(crudItemPath("/v1/w", "id-1", "unarchive")).toBe("/v1/w/id-1/unarchive");
    expect(crudItemPath("/v1/w", "id-1", "trash")).toBe("/v1/w/id-1/trash");
    expect(crudItemPath("/v1/w", "id-1", "restore")).toBe("/v1/w/id-1/restore");
    expect(crudItemPath("/v1/w", "id-1", "purge")).toBe("/v1/w/id-1/purge");
  });
});
