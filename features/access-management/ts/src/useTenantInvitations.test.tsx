import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AccessInvitation, AccessManagementClient } from "./types";
import { useTenantInvitations } from "./useTenantInvitations";

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

describe("useTenantInvitations", () => {
  it("ignores a late response from the previous tenant", async () => {
    const tenantA = deferred<AccessInvitation[]>();
    const tenantB = deferred<AccessInvitation[]>();
    const client = {
      listUsers: vi.fn(),
      listTenants: vi.fn(),
      listInvitations: vi.fn((tenantId: string) =>
        tenantId === "tenant-a" ? tenantA.promise : tenantB.promise
      ),
    } satisfies AccessManagementClient;

    const { result, rerender } = renderHook(
      ({ tenantId }) => useTenantInvitations(client, tenantId),
      { initialProps: { tenantId: "tenant-a" as string | null } }
    );

    rerender({ tenantId: "tenant-b" });
    act(() => {
      tenantA.resolve([
        {
          id: "invite-a",
          tenantId: "tenant-a",
          email: "a@example.com",
          role: "viewer",
          status: "pending",
          expiresAt: "2030-01-01T00:00:00Z",
          createdAt: "2029-01-01T00:00:00Z",
        },
      ]);
    });
    expect(result.current.rows).toEqual([]);

    act(() => {
      tenantB.resolve([
        {
          id: "invite-b",
          tenantId: "tenant-b",
          email: "b@example.com",
          role: "viewer",
          status: "pending",
          expiresAt: "2030-01-01T00:00:00Z",
          createdAt: "2029-01-01T00:00:00Z",
        },
      ]);
    });
    await waitFor(() => expect(result.current.rows.map((row) => row.id)).toEqual(["invite-b"]));
  });
});
