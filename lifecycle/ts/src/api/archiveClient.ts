import type {
  ArchiveRequest,
  Fetcher,
  HardDeleteRequest,
  PathResolver,
  RestoreRequest,
} from "../types";

/**
 * Thin client over the consumer-supplied Fetcher + PathResolver. The library
 * never hardcodes a base URL or a REST convention — pymes supplies them.
 */
export class ArchiveClient {
  constructor(
    private readonly fetcher: Fetcher,
    private readonly resolvePath: PathResolver,
  ) {}

  archive(req: ArchiveRequest): Promise<unknown> {
    const url = this.resolvePath(req.resourceType, req.resourceId, "archive");
    return this.fetcher(url, {
      method: "POST",
      body: JSON.stringify({
        reason: req.reason,
        actor: req.actor,
        tenant_id: req.tenantId,
        batch_id: req.batchId,
      }),
    });
  }

  restore(req: RestoreRequest): Promise<unknown> {
    const url = this.resolvePath(req.resourceType, req.resourceId, "restore");
    return this.fetcher(url, {
      method: "POST",
      body: JSON.stringify({
        reason: req.reason,
        actor: req.actor,
        tenant_id: req.tenantId,
      }),
    });
  }

  hardDelete(req: HardDeleteRequest): Promise<unknown> {
    const url = this.resolvePath(req.resourceType, req.resourceId, "hard");
    return this.fetcher(url, {
      method: "DELETE",
      body: JSON.stringify({
        reason: req.reason,
        actor: req.actor,
        tenant_id: req.tenantId,
        must_be_archived: req.mustBeArchived ?? true,
      }),
    });
  }

  /** GET the archived listing for a resource type. */
  listArchived(resourceType: string): Promise<unknown> {
    // The PathResolver decides the listing path. Caller passes empty resourceId.
    const url = this.resolvePath(resourceType, "", "archived");
    return this.fetcher(url, { method: "GET" });
  }
}
