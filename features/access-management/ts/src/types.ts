export type AccessTab = "users" | "tenants" | "invitations";
export type ResourceState = "active" | "archived" | "trash";
export type InvitationStatus = "pending" | "accepted" | "revoked" | "expired";
export type DeliveryStatus = "sent" | "queued" | "failed" | "disabled" | "unknown";

export interface TenantRef {
  id: string;
  name: string;
}

export interface AccessPolicy {
  canManageUsers: boolean;
  canManageTenants: boolean;
  canManageInvitations: boolean;
}

export interface AccessContext {
  userId: string;
  activeTenant: TenantRef | null;
  manageableTenants: TenantRef[];
  policy: AccessPolicy;
}

export interface Membership {
  tenantId: string;
  tenantName: string;
  role: string;
  status?: string;
}

export interface AccessUser {
  id: string;
  email: string;
  username: string | null;
  memberships?: Membership[];
  archivedAt?: string | null;
  deletedAt?: string | null;
}

export interface AccessTenant extends TenantRef {
  status?: "active" | "suspended" | string;
  createdAt?: string;
  archivedAt?: string | null;
  deletedAt?: string | null;
}

export interface AccessInvitation {
  id: string;
  tenantId: string;
  email: string;
  role: string;
  status: InvitationStatus | string;
  expiresAt: string;
  createdAt: string;
  emailSent?: boolean;
  deliveryStatus?: DeliveryStatus;
}

export interface CreateUserInput {
  email: string;
  password: string;
  username?: string | null;
}

export interface UpdateUserInput {
  email?: string;
  /**
   * Omitted keeps the username. `null` removes it.
   */
  username?: string | null;
}

export interface CreateInvitationInput {
  email: string;
  role: string;
}

export interface AccessManagementClient {
  listUsers(state?: ResourceState): Promise<AccessUser[]>;
  listTenants(state?: ResourceState): Promise<AccessTenant[]>;
  listInvitations(tenantId: string): Promise<AccessInvitation[]>;
  createUser?(input: CreateUserInput): Promise<void>;
  updateUser?(userId: string, input: UpdateUserInput): Promise<void>;
  createTenant?(name: string): Promise<void>;
  updateTenant?(tenantId: string, name: string): Promise<void>;
  createInvitation?(tenantId: string, input: CreateInvitationInput): Promise<void>;
  revokeInvitation?(invitationId: string): Promise<void>;
  resendInvitation?(invitationId: string): Promise<void>;
}

export interface AccessManagementLabels {
  users: string;
  tenants: string;
  invitations: string;
  search: string;
  loading: string;
  retry: string;
  noResults: string;
  unavailable: string;
  newUser: string;
  newTenant: string;
  newInvitation: string;
  email: string;
  username: string;
  optional: string;
  password: string;
  showPassword: string;
  hidePassword: string;
  role: string;
  status: string;
  expires: string;
  memberships: string;
  save: string;
  cancel: string;
  edit: string;
  revoke: string;
  resend: string;
  tenantName: string;
}
