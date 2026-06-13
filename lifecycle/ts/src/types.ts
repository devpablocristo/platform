/**
 * Domain-agnostic types for the canonical resource lifecycle.
 *
 * Archive is not deletion. Archived resources are retained but excluded from
 * active workflows by default. Trash is reversible delete. Purge is
 * irreversible deletion.
 */

export type LifecycleState = "active" | "archived" | "trashed" | "purged";

export function participatesInAutomation(state: LifecycleState | ""): boolean {
  return state === "" || state === "active";
}

export type ArchiveRequest = {
  resourceType: string;
  resourceId: string;
  tenantId: string;
  actor: string;
  reason?: string;
  batchId?: string;
};

export type UnarchiveRequest = {
  resourceType: string;
  resourceId: string;
  tenantId: string;
  actor: string;
  reason?: string;
};

export type TrashRequest = {
  resourceType: string;
  resourceId: string;
  tenantId: string;
  actor: string;
  reason?: string;
  batchId?: string;
};

export type RestoreRequest = {
  resourceType: string;
  resourceId: string;
  tenantId: string;
  actor: string;
  reason?: string;
};

export type PurgeRequest = {
  resourceType: string;
  resourceId: string;
  tenantId: string;
  actor: string;
  reason?: string;
  /** When true, refuses to purge a resource that is not trashed yet. */
  mustBeTrashed?: boolean;
};

export type LifecyclePolicy = {
  resourceType: string;
  allowArchive: boolean;
  allowTrash: boolean;
  allowPurge: boolean;
  requireReason: boolean;
  /** 0 = retain forever in trash; >0 = purge after this many days. */
  retentionDays: number;
};

/** Path segments mirrored from platform/lifecycle/go/paths. */
export const PathSegments = {
  archived: "archived",
  archive: "archive",
  unarchive: "unarchive",
  trash: "trash",
  restore: "restore",
  purge: "purge",
} as const;

export type PathSegment = keyof typeof PathSegments;

export type PathResolver = (
  resourceType: string,
  resourceId: string,
  segment: PathSegment,
) => string;

export type Fetcher = (
  url: string,
  init?: RequestInit & { method?: "GET" | "POST" | "DELETE" | "PUT" | "PATCH" },
) => Promise<unknown>;

export type ArchiveLabels = {
  title: string;
  description?: string;
  confirmButton: string;
  cancelButton: string;
  reasonLabel?: string;
  reasonPlaceholder?: string;
  reasonRequiredHint?: string;
};

export type RetentionLabels = {
  title: string;
  daysLabel: string;
  allowPurgeLabel: string;
  requireReasonLabel: string;
  saveButton: string;
};

export type BulkArchiveLabels = {
  selectionPrefix: string;
  archiveButton: string;
  cancelButton: string;
  reasonLabel?: string;
  reasonPlaceholder?: string;
};
