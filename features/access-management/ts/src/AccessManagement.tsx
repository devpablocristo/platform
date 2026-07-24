import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";

import { buildCreateUserInput, buildUsernameUpdate, validateEmail } from "./forms";
import { accessTabsFor } from "./policy";
import { matchesAccessSearch, normalizeAccessSearch } from "./search";
import type {
  AccessContext,
  AccessManagementClient,
  AccessManagementLabels,
  AccessTab,
  AccessTenant,
  AccessUser,
} from "./types";
import { useTenantInvitations } from "./useTenantInvitations";

const DEFAULT_LABELS: AccessManagementLabels = {
  users: "Users",
  tenants: "Tenants",
  invitations: "Invitations",
  search: "Search",
  loading: "Loading…",
  retry: "Retry",
  noResults: "No results",
  unavailable: "You do not have access to this section.",
  newUser: "New user",
  newTenant: "New tenant",
  newInvitation: "New invitation",
  email: "Email",
  username: "Username",
  optional: "optional",
  password: "Password",
  showPassword: "Show password",
  hidePassword: "Hide password",
  role: "Role",
  status: "Status",
  expires: "Expires",
  memberships: "Memberships",
  save: "Save",
  cancel: "Cancel",
  edit: "Edit",
  revoke: "Revoke",
  resend: "Resend",
  tenantName: "Tenant name",
};

export interface AccessManagementProps {
  client: AccessManagementClient;
  context: AccessContext;
  labels?: Partial<AccessManagementLabels>;
  roles?: readonly string[];
  className?: string;
  onError?: (error: Error) => void;
}

type DialogState =
  | { kind: "user-create" }
  | { kind: "user-edit"; user: AccessUser }
  | { kind: "tenant-create" }
  | { kind: "tenant-edit"; tenant: AccessTenant }
  | { kind: "invitation-create" }
  | null;

function errorOf(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error));
}

function tabLabel(tab: AccessTab, labels: AccessManagementLabels): string {
  if (tab === "users") return labels.users;
  if (tab === "tenants") return labels.tenants;
  return labels.invitations;
}

export function AccessManagement({
  client,
  context,
  labels: labelOverrides,
  roles = ["owner", "admin", "manager", "viewer"],
  className = "",
  onError,
}: AccessManagementProps) {
  const labels = useMemo(() => ({ ...DEFAULT_LABELS, ...labelOverrides }), [labelOverrides]);
  const tabs = useMemo(() => accessTabsFor(context.policy), [context.policy]);
  const [tab, setTab] = useState<AccessTab | null>(tabs[0] ?? null);
  const [query, setQuery] = useState("");
  const [users, setUsers] = useState<AccessUser[]>([]);
  const [tenants, setTenants] = useState<AccessTenant[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [dialog, setDialog] = useState<DialogState>(null);
  const invitations = useTenantInvitations(client, context.activeTenant?.id ?? null);

  useEffect(() => {
    setTab((current) => (current && tabs.includes(current) ? current : (tabs[0] ?? null)));
  }, [tabs]);

  useEffect(() => {
    setDialog(null);
  }, [context.userId, context.activeTenant?.id]);

  const load = useCallback(async () => {
    if (!tab || tab === "invitations") {
      invitations.reload();
      return;
    }
    setLoading(true);
    setError(null);
    try {
      if (tab === "users") setUsers(await client.listUsers("active"));
      if (tab === "tenants") setTenants(await client.listTenants("active"));
    } catch (cause) {
      const nextError = errorOf(cause);
      setError(nextError);
      onError?.(nextError);
    } finally {
      setLoading(false);
    }
  }, [client, invitations.reload, onError, tab]);

  useEffect(() => {
    void load();
  }, [load]);

  const normalizedQuery = normalizeAccessSearch(query);
  const visibleUsers = users.filter((user) =>
    matchesAccessSearch(normalizedQuery, [
      user.email,
      user.username,
      ...(user.memberships ?? []).flatMap((membership) => [
        membership.tenantName,
        membership.role,
      ]),
    ])
  );
  const visibleTenants = tenants.filter((tenant) =>
    matchesAccessSearch(normalizedQuery, [tenant.name, tenant.status])
  );
  const visibleInvitations = invitations.rows.filter((invitation) =>
    matchesAccessSearch(normalizedQuery, [
      invitation.email,
      invitation.role,
      invitation.status,
      invitation.deliveryStatus,
    ])
  );
  const activeError = tab === "invitations" ? invitations.error : error;
  const activeLoading = tab === "invitations" ? invitations.loading : loading;

  if (!tab) {
    return <div className={`platform-access platform-access--empty ${className}`}>{labels.unavailable}</div>;
  }

  return (
    <section className={`platform-access ${className}`}>
      <header className="platform-access__bar">
        <nav className="platform-access__tabs" aria-label="Access management">
          {tabs.map((item) => (
            <button
              className={`platform-access__tab ${tab === item ? "is-active" : ""}`}
              key={item}
              onClick={() => setTab(item)}
              type="button"
            >
              {tabLabel(item, labels)}
            </button>
          ))}
        </nav>
        <label className="platform-access__search">
          <span aria-hidden="true">⌕</span>
          <input
            aria-label={labels.search}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={labels.search}
            type="search"
            value={query}
          />
        </label>
      </header>

      {activeError ? (
        <div className="platform-access__feedback" role="alert">
          <span>{activeError.message}</span>
          <button type="button" onClick={() => void load()}>{labels.retry}</button>
        </div>
      ) : null}

      {tab === "users" ? (
        <Panel
          action={client.createUser ? labels.newUser : undefined}
          loading={activeLoading}
          loadingLabel={labels.loading}
          onAction={() => setDialog({ kind: "user-create" })}
          title={labels.users}
        >
          <table>
            <thead><tr><th>{labels.email}</th><th>{labels.username}</th><th>{labels.memberships}</th><th /></tr></thead>
            <tbody>
              {visibleUsers.map((user) => (
                <tr key={user.id}>
                  <td>{user.email}</td>
                  <td>{user.username || "—"}</td>
                  <td>{(user.memberships ?? []).map((membership) => `${membership.tenantName}: ${membership.role}`).join(", ") || "—"}</td>
                  <td>{client.updateUser ? <button type="button" onClick={() => setDialog({ kind: "user-edit", user })}>{labels.edit}</button> : null}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!activeLoading && visibleUsers.length === 0 ? <Empty>{labels.noResults}</Empty> : null}
        </Panel>
      ) : null}

      {tab === "tenants" ? (
        <Panel
          action={client.createTenant ? labels.newTenant : undefined}
          loading={activeLoading}
          loadingLabel={labels.loading}
          onAction={() => setDialog({ kind: "tenant-create" })}
          title={labels.tenants}
        >
          <table>
            <thead><tr><th>{labels.tenantName}</th><th>{labels.status}</th><th /></tr></thead>
            <tbody>
              {visibleTenants.map((tenant) => (
                <tr key={tenant.id}>
                  <td>{tenant.name}</td><td>{tenant.status || "—"}</td>
                  <td>{client.updateTenant ? <button type="button" onClick={() => setDialog({ kind: "tenant-edit", tenant })}>{labels.edit}</button> : null}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!activeLoading && visibleTenants.length === 0 ? <Empty>{labels.noResults}</Empty> : null}
        </Panel>
      ) : null}

      {tab === "invitations" ? (
        <Panel
          action={context.activeTenant && client.createInvitation ? labels.newInvitation : undefined}
          loading={activeLoading}
          loadingLabel={labels.loading}
          onAction={() => setDialog({ kind: "invitation-create" })}
          title={labels.invitations}
        >
          <table>
            <thead><tr><th>{labels.email}</th><th>{labels.role}</th><th>{labels.status}</th><th>{labels.expires}</th><th /></tr></thead>
            <tbody>
              {visibleInvitations.map((invitation) => (
                <tr key={invitation.id}>
                  <td>{invitation.email}</td><td>{invitation.role}</td><td>{invitation.status}</td>
                  <td>{new Date(invitation.expiresAt).toLocaleDateString()}</td>
                  <td className="platform-access__actions">
                    {client.resendInvitation ? <button type="button" onClick={() => void mutate(() => client.resendInvitation!(invitation.id), invitations.reload, onError)}>{labels.resend}</button> : null}
                    {client.revokeInvitation && invitation.status === "pending" ? <button type="button" onClick={() => void mutate(() => client.revokeInvitation!(invitation.id), invitations.reload, onError)}>{labels.revoke}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!activeLoading && visibleInvitations.length === 0 ? <Empty>{labels.noResults}</Empty> : null}
        </Panel>
      ) : null}

      {dialog ? (
        <AccessDialog
          client={client}
          context={context}
          dialog={dialog}
          labels={labels}
          onClose={() => setDialog(null)}
          onError={onError}
          onSaved={() => {
            setDialog(null);
            void load();
          }}
          roles={roles}
        />
      ) : null}
    </section>
  );
}

async function mutate(action: () => Promise<void>, reload: () => void, onError?: (error: Error) => void) {
  try {
    await action();
    reload();
  } catch (cause) {
    onError?.(errorOf(cause));
  }
}

function Panel({ action, children, loading, loadingLabel, onAction, title }: {
  action?: string;
  children: React.ReactNode;
  loading: boolean;
  loadingLabel: string;
  onAction: () => void;
  title: string;
}) {
  return (
    <div className="platform-access__panel">
      <div className="platform-access__heading">
        <h2>{title}</h2>
        {action ? <button className="platform-access__primary" type="button" onClick={onAction}>+ {action}</button> : null}
      </div>
      {loading ? <div className="platform-access__loading" aria-live="polite">{loadingLabel}</div> : children}
    </div>
  );
}

function Empty({ children }: { children: React.ReactNode }) {
  return <div className="platform-access__empty">{children}</div>;
}

function AccessDialog({ client, context, dialog, labels, onClose, onError, onSaved, roles }: {
  client: AccessManagementClient;
  context: AccessContext;
  dialog: Exclude<DialogState, null>;
  labels: AccessManagementLabels;
  onClose: () => void;
  onError?: (error: Error) => void;
  onSaved: () => void;
  roles: readonly string[];
}) {
  const user = dialog.kind === "user-edit" ? dialog.user : null;
  const tenant = dialog.kind === "tenant-edit" ? dialog.tenant : null;
  const [email, setEmail] = useState(user?.email ?? "");
  const [username, setUsername] = useState(user?.username ?? "");
  const [password, setPassword] = useState("");
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [name, setName] = useState(tenant?.name ?? "");
  const [role, setRole] = useState(roles[roles.length - 1] ?? "");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setFormError(null);
    try {
      if (dialog.kind === "user-create") {
        await client.createUser?.(buildCreateUserInput({ email, username, password }));
      } else if (dialog.kind === "user-edit") {
        if (!validateEmail(email)) throw new Error("A valid email is required");
        await client.updateUser?.(dialog.user.id, {
          ...(email.trim().toLocaleLowerCase() !== dialog.user.email ? { email: email.trim().toLocaleLowerCase() } : {}),
          ...buildUsernameUpdate(dialog.user.username, username),
        });
      } else if (dialog.kind === "tenant-create") {
        if (!name.trim()) throw new Error(`${labels.tenantName} is required`);
        await client.createTenant?.(name.trim());
      } else if (dialog.kind === "tenant-edit") {
        if (!name.trim()) throw new Error(`${labels.tenantName} is required`);
        await client.updateTenant?.(dialog.tenant.id, name.trim());
      } else {
        if (!context.activeTenant) throw new Error("An active tenant is required");
        if (!validateEmail(email)) throw new Error("A valid email is required");
        await client.createInvitation?.(context.activeTenant.id, { email: email.trim().toLocaleLowerCase(), role });
      }
      onSaved();
    } catch (cause) {
      const nextError = errorOf(cause);
      setFormError(nextError.message);
      onError?.(nextError);
    } finally {
      setSaving(false);
    }
  }

  const title =
    dialog.kind === "user-create" ? labels.newUser :
    dialog.kind === "user-edit" ? labels.edit :
    dialog.kind === "tenant-create" ? labels.newTenant :
    dialog.kind === "tenant-edit" ? labels.edit :
    labels.newInvitation;

  return (
    <div className="platform-access__backdrop" role="presentation" onMouseDown={onClose}>
      <div className="platform-access__dialog" role="dialog" aria-modal="true" aria-label={title} onMouseDown={(event) => event.stopPropagation()}>
        <h2>{title}</h2>
        <form onSubmit={(event) => void submit(event)}>
          {dialog.kind.startsWith("user") || dialog.kind === "invitation-create" ? (
            <label>{labels.email}<input required type="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label>
          ) : null}
          {dialog.kind.startsWith("user") ? (
            <label>{labels.username} ({labels.optional})<input value={username} onChange={(event) => setUsername(event.target.value)} /></label>
          ) : null}
          {dialog.kind === "user-create" ? (
            <label>
              {labels.password}
              <span className="platform-access__password">
                <input required minLength={8} type={passwordVisible ? "text" : "password"} value={password} onChange={(event) => setPassword(event.target.value)} />
                <button type="button" aria-label={passwordVisible ? labels.hidePassword : labels.showPassword} onClick={() => setPasswordVisible((visible) => !visible)}>
                  {passwordVisible ? "◉" : "◎"}
                </button>
              </span>
            </label>
          ) : null}
          {dialog.kind.startsWith("tenant") ? (
            <label>{labels.tenantName}<input required value={name} onChange={(event) => setName(event.target.value)} /></label>
          ) : null}
          {dialog.kind === "invitation-create" ? (
            <label>{labels.role}<select value={role} onChange={(event) => setRole(event.target.value)}>{roles.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
          ) : null}
          {formError ? <p className="platform-access__form-error" role="alert">{formError}</p> : null}
          <div className="platform-access__dialog-actions">
            <button type="button" onClick={onClose}>{labels.cancel}</button>
            <button className="platform-access__primary" disabled={saving} type="submit">{labels.save}</button>
          </div>
        </form>
      </div>
    </div>
  );
}
