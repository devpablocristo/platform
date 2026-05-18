// @vitest-environment node

import { describe, expect, it, vi } from "vitest";
import { createPublicSchedulingClient, createSchedulingClient } from "./client";

describe("scheduling client compatibility", () => {
  it("normalizes deprecated appointments_enabled into scheduling_enabled for public info", async () => {
    const request = vi.fn().mockResolvedValue({
      org_id: "org-demo",
      slug: "demo-org",
      name: "Demo Org",
      business_name: "Demo Scheduling",
      appointments_enabled: true,
    });

    const client = createPublicSchedulingClient(request);
    const info = await client.getBusinessInfo("demo-org");

    expect(request).toHaveBeenCalledWith("/v1/public/demo-org/info");
    expect(info.scheduling_enabled).toBe(true);
    expect(info.appointments_enabled).toBe(true);
  });

  it("builds scheduling slot queries with branch, service and resource selectors", async () => {
    const request = vi.fn().mockResolvedValue({ items: [] });

    const client = createSchedulingClient(request);
    await client.listSlots({
      branchId: "branch-1",
      serviceId: "service-1",
      resourceId: "resource-1",
      date: "2026-04-03",
    });

    expect(request).toHaveBeenCalledWith(
      "/v1/scheduling/slots?branch_id=branch-1&service_id=service-1&resource_id=resource-1&date=2026-04-03",
    );
  });

  it("lists blocked ranges with branch, resource and date filters", async () => {
    const request = vi.fn().mockResolvedValue({ items: [] });

    const client = createSchedulingClient(request);
    await client.listBlockedRanges({ branchId: "branch-1", resourceId: "res-1", date: "2026-04-06" });

    expect(request).toHaveBeenCalledWith(
      "/v1/scheduling/blocked-ranges?branch_id=branch-1&resource_id=res-1&date=2026-04-06",
    );
  });

  it("creates a blocked range with POST and the payload as body", async () => {
    const request = vi.fn().mockResolvedValue({ id: "br-1" });

    const client = createSchedulingClient(request);
    await client.createBlockedRange({
      branch_id: "branch-1",
      kind: "manual",
      reason: "Reunión con proveedor",
      start_at: "2026-04-06T17:00:00Z",
      end_at: "2026-04-06T18:00:00Z",
    });

    expect(request).toHaveBeenCalledWith("/v1/scheduling/blocked-ranges", {
      method: "POST",
      body: {
        branch_id: "branch-1",
        kind: "manual",
        reason: "Reunión con proveedor",
        start_at: "2026-04-06T17:00:00Z",
        end_at: "2026-04-06T18:00:00Z",
      },
    });
  });

  it("updates a blocked range with PATCH and the id in the path", async () => {
    const request = vi.fn().mockResolvedValue({ id: "br-1" });

    const client = createSchedulingClient(request);
    await client.updateBlockedRange("br-1", {
      branch_id: "branch-1",
      kind: "manual",
      start_at: "2026-04-06T17:00:00Z",
      end_at: "2026-04-06T19:00:00Z",
    });

    expect(request).toHaveBeenCalledWith("/v1/scheduling/blocked-ranges/br-1", {
      method: "PATCH",
      body: {
        branch_id: "branch-1",
        kind: "manual",
        start_at: "2026-04-06T17:00:00Z",
        end_at: "2026-04-06T19:00:00Z",
      },
    });
  });

  it("deletes a blocked range with DELETE", async () => {
    const request = vi.fn().mockResolvedValue(undefined);

    const client = createSchedulingClient(request);
    await client.deleteBlockedRange("br-1");

    expect(request).toHaveBeenCalledWith("/v1/scheduling/blocked-ranges/br-1", { method: "DELETE" });
  });
});
