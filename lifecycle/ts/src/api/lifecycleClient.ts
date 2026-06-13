import type {
  ArchiveRequest,
  Fetcher,
  PathResolver,
  PurgeRequest,
  RestoreRequest,
  TrashRequest,
  UnarchiveRequest,
} from "../types";

/**
 * Thin client over a consumer-supplied Fetcher + PathResolver. The library
 * never hardcodes a base URL or product REST convention.
 */
export class LifecycleClient {
  constructor(
    private readonly fetcher: Fetcher,
    private readonly resolvePath: PathResolver,
  ) {}

  archive(req: ArchiveRequest): Promise<unknown> {
    return this.post(this.resolvePath(req.resourceType, req.resourceId, "archive"), {
      reason: req.reason,
      actor: req.actor,
      tenant_id: req.tenantId,
      batch_id: req.batchId,
    });
  }

  unarchive(req: UnarchiveRequest): Promise<unknown> {
    return this.post(this.resolvePath(req.resourceType, req.resourceId, "unarchive"), {
      reason: req.reason,
      actor: req.actor,
      tenant_id: req.tenantId,
    });
  }

  trash(req: TrashRequest): Promise<unknown> {
    return this.post(this.resolvePath(req.resourceType, req.resourceId, "trash"), {
      reason: req.reason,
      actor: req.actor,
      tenant_id: req.tenantId,
      batch_id: req.batchId,
    });
  }

  restore(req: RestoreRequest): Promise<unknown> {
    return this.post(this.resolvePath(req.resourceType, req.resourceId, "restore"), {
      reason: req.reason,
      actor: req.actor,
      tenant_id: req.tenantId,
    });
  }

  purge(req: PurgeRequest): Promise<unknown> {
    return this.fetcher(this.resolvePath(req.resourceType, req.resourceId, "purge"), {
      method: "DELETE",
      body: JSON.stringify({
        reason: req.reason,
        actor: req.actor,
        tenant_id: req.tenantId,
        must_be_trashed: req.mustBeTrashed ?? true,
      }),
    });
  }

  listArchived(resourceType: string): Promise<unknown> {
    return this.fetcher(this.resolvePath(resourceType, "", "archived"), { method: "GET" });
  }

  listTrash(resourceType: string): Promise<unknown> {
    return this.fetcher(this.resolvePath(resourceType, "", "trash"), { method: "GET" });
  }

  private post(url: string, body: Record<string, unknown>): Promise<unknown> {
    return this.fetcher(url, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }
}
