import type { AccessPolicy, AccessTab } from "./types";

export function accessTabsFor(policy: AccessPolicy): AccessTab[] {
  const tabs: AccessTab[] = [];
  if (policy.canManageUsers) tabs.push("users");
  if (policy.canManageTenants) tabs.push("tenants");
  if (policy.canManageInvitations) tabs.push("invitations");
  return tabs;
}
