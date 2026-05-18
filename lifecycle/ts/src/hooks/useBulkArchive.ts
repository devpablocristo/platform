import { useCallback, useState } from "react";
import type { ArchiveClient } from "../api/archiveClient";

export type BulkArchiveOutcome = {
  resourceId: string;
  ok: boolean;
  error?: Error;
};

export type UseBulkArchiveResult = {
  bulkArchive: (params: {
    resourceType: string;
    resourceIds: string[];
    tenantId: string;
    actor: string;
    reason?: string;
  }) => Promise<BulkArchiveOutcome[]>;
  isArchiving: boolean;
};

/**
 * Sequential bulk archive: archives each ID one at a time, collecting per-ID
 * outcomes. Useful for UIs that want to show a progress bar and report
 * partial failures. For high-throughput batch jobs, the consumer can wire a
 * server-side BulkArchive endpoint and call client.fetcher directly.
 */
export function useBulkArchive(client: ArchiveClient): UseBulkArchiveResult {
  const [isArchiving, setIsArchiving] = useState(false);

  const bulkArchive = useCallback(
    async (params: {
      resourceType: string;
      resourceIds: string[];
      tenantId: string;
      actor: string;
      reason?: string;
    }) => {
      setIsArchiving(true);
      const batchId = generateBatchId();
      const outcomes: BulkArchiveOutcome[] = [];
      try {
        for (const id of params.resourceIds) {
          try {
            await client.archive({
              resourceType: params.resourceType,
              resourceId: id,
              tenantId: params.tenantId,
              actor: params.actor,
              reason: params.reason,
              batchId,
            });
            outcomes.push({ resourceId: id, ok: true });
          } catch (err) {
            outcomes.push({
              resourceId: id,
              ok: false,
              error: err instanceof Error ? err : new Error(String(err)),
            });
          }
        }
      } finally {
        setIsArchiving(false);
      }
      return outcomes;
    },
    [client],
  );

  return { bulkArchive, isArchiving };
}

function generateBatchId(): string {
  // Avoid pulling in crypto deps; uuid v4-ish from crypto.randomUUID() if
  // available, otherwise a coarser timestamp-based identifier.
  const g = (globalThis as { crypto?: { randomUUID?: () => string } }).crypto;
  if (g?.randomUUID) return g.randomUUID();
  return `batch-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}
