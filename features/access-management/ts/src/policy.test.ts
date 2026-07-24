import { describe, expect, it } from "vitest";

import { accessTabsFor } from "./policy";

describe("access tab policy", () => {
  it("keeps each management capability independent", () => {
    expect(
      accessTabsFor({
        canManageUsers: false,
        canManageTenants: false,
        canManageInvitations: true,
      })
    ).toEqual(["invitations"]);
  });
});
