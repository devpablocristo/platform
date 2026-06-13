import { describe, expect, it, vi } from "vitest";
import { LifecycleClient } from "./lifecycleClient";
import type { PathSegment } from "../types";

describe("LifecycleClient", () => {
  it("calls archive, unarchive, trash, restore and purge paths", async () => {
    const calls: Array<{ url: string; init?: RequestInit }> = [];
    const fetcher = vi.fn(async (url: string, init?: RequestInit) => {
      calls.push({ url, init });
      return {};
    });
    const resolver = (resourceType: string, resourceId: string, segment: PathSegment) =>
      `/v1/${resourceType}/${resourceId || "_collection"}/${segment}`;
    const client = new LifecycleClient(fetcher, resolver);

    await client.archive(base("archive"));
    await client.unarchive(base("unarchive"));
    await client.trash(base("trash"));
    await client.restore(base("restore"));
    await client.purge({ ...base("purge"), mustBeTrashed: true });

    expect(calls.map((c) => c.url)).toEqual([
      "/v1/documents/id-1/archive",
      "/v1/documents/id-1/unarchive",
      "/v1/documents/id-1/trash",
      "/v1/documents/id-1/restore",
      "/v1/documents/id-1/purge",
    ]);
    expect(calls.map((c) => c.init?.method)).toEqual(["POST", "POST", "POST", "POST", "DELETE"]);
    expect(JSON.parse(String(calls[4]?.init?.body))).toMatchObject({
      must_be_trashed: true,
      tenant_id: "org-1",
    });
  });

  it("lists archived and trash collections", async () => {
    const fetcher = vi.fn(async () => ({}));
    const resolver = (resourceType: string, resourceId: string, segment: PathSegment) =>
      `/v1/${resourceType}/${resourceId || segment}`;
    const client = new LifecycleClient(fetcher, resolver);

    await client.listArchived("documents");
    await client.listTrash("documents");

    expect(fetcher).toHaveBeenNthCalledWith(1, "/v1/documents/archived", { method: "GET" });
    expect(fetcher).toHaveBeenNthCalledWith(2, "/v1/documents/trash", { method: "GET" });
  });
});

function base(actor: string) {
  return {
    resourceType: "documents",
    resourceId: "id-1",
    tenantId: "org-1",
    actor,
    reason: "test",
  };
}
