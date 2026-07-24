import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AccessManagement } from "./AccessManagement";
import type { AccessContext, AccessManagementClient } from "./types";

function context(activeTenantId = "tenant-a"): AccessContext {
  return {
    userId: "user-1",
    activeTenant: { id: activeTenantId, name: activeTenantId },
    manageableTenants: [{ id: activeTenantId, name: activeTenantId }],
    policy: {
      canManageUsers: true,
      canManageTenants: true,
      canManageInvitations: true,
    },
  };
}

function client(): AccessManagementClient {
  return {
    listUsers: vi.fn().mockResolvedValue([
      { id: "u1", email: "ana@example.com", username: null },
    ]),
    listTenants: vi.fn().mockResolvedValue([{ id: "tenant-a", name: "Acme" }]),
    listInvitations: vi.fn().mockResolvedValue([]),
    createUser: vi.fn().mockResolvedValue(undefined),
    createInvitation: vi.fn().mockResolvedValue(undefined),
  };
}

describe("AccessManagement", () => {
  it("uses one global search across tabs", async () => {
    render(<AccessManagement client={client()} context={context()} />);
    await screen.findByText("ana@example.com");
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "missing" } });
    expect(screen.getByText("No results")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Tenants" }));
    expect(screen.getByRole<HTMLInputElement>("searchbox").value).toBe("missing");
  });

  it("creates invitations only for the active tenant and has no tenant selector", async () => {
    const api = client();
    render(<AccessManagement client={api} context={context("tenant-current")} />);
    fireEvent.click(screen.getByRole("button", { name: "Invitations" }));
    fireEvent.click(await screen.findByRole("button", { name: /\+ New invitation/ }));
    expect(screen.queryByLabelText(/tenant/i)).toBeNull();
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "invite@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.createInvitation).toHaveBeenCalledWith("tenant-current", {
        email: "invite@example.com",
        role: "viewer",
      })
    );
  });

  it("exposes an accessible password visibility toggle", async () => {
    render(<AccessManagement client={client()} context={context()} />);
    fireEvent.click(await screen.findByRole("button", { name: /\+ New user/ }));
    const password = screen.getByLabelText("Password");
    expect(password.getAttribute("type")).toBe("password");
    fireEvent.click(screen.getByRole("button", { name: "Show password" }));
    expect(password.getAttribute("type")).toBe("text");
    expect(screen.getByRole("button", { name: "Hide password" })).toBeTruthy();
  });
});
