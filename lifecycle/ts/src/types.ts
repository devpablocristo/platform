/**
 * Domain-agnostic types for the CRUDAR (Create-Read-Update-Delete-Archive-Restore)
 * lifecycle. `resourceType` and `actor` are opaque strings — the consumer
 * (e.g. pymes) supplies the vocabulary. This package never knows entity names.
 *
 * See platform/docs/migration plan § Invariantes de agnosticidad for the
 * design constraints.
 */

export type ArchiveRequest = {
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

export type HardDeleteRequest = {
  resourceType: string;
  resourceId: string;
  tenantId: string;
  actor: string;
  reason?: string;
  /** When true, refuses to hard-delete a resource that is not archived yet. */
  mustBeArchived?: boolean;
};

/**
 * Mirrors the Go ArchivePolicy mechanism. Instances of this struct are
 * declared by the consumer (the policy describes a *kind* of resource, the
 * library uses them to decide whether to allow an action and how to validate
 * inputs).
 */
export type ArchivePolicy = {
  resourceType: string;
  allowArchive: boolean;
  allowHardDelete: boolean;
  /** When true, ArchiveRequest.reason must be non-empty. */
  requireReason: boolean;
  /** 0 = retain forever; >0 = purge after this many days. */
  retentionDays: number;
};

/** Path segments mirrored from platform/lifecycle/go/paths. */
export const PathSegments = {
  archived: "archived",
  archive: "archive",
  restore: "restore",
  hard: "hard",
} as const;

export type PathSegment = keyof typeof PathSegments;

/**
 * Resolves a REST path for an action on a specific resource.
 *
 * The library does not assume any base URL convention — the consumer
 * provides a PathResolver. Typical pymes resolver:
 *
 *   const resolver: PathResolver = (resourceType, resourceId, segment) => {
 *     // pymes uses pluralized REST paths, e.g. /v1/widgets/{id}/archive
 *     const collection = pluralize(resourceType);
 *     return segment === "archived"
 *       ? `/v1/${collection}/archived`
 *       : `/v1/${collection}/${resourceId}/${segment}`;
 *   };
 */
export type PathResolver = (
  resourceType: string,
  resourceId: string,
  segment: PathSegment,
) => string;

/** Minimal HTTP fetcher abstraction. Caller injects implementation. */
export type Fetcher = (
  url: string,
  init?: RequestInit & { method?: "GET" | "POST" | "DELETE" | "PUT" | "PATCH" },
) => Promise<unknown>;

/** Localizable labels — the consumer supplies the strings (§ Invariante I7). */
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
  allowHardDeleteLabel: string;
  requireReasonLabel: string;
  saveButton: string;
};

export type BulkArchiveLabels = {
  selectionPrefix: string;       // "{n} selected" — `{n}` is replaced
  archiveButton: string;
  cancelButton: string;
  reasonLabel?: string;
  reasonPlaceholder?: string;
};
