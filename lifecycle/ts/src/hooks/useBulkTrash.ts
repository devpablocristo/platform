import { useCallback, useState } from "react";
import type { LifecycleClient } from "../api/lifecycleClient";

export type BulkTrashOutcome = {
  resourceId: string;
  ok: boolean;
  error?: Error;
};

export type UseBulkTrashResult = {
  bulkTrash: (params: {
    resourceType: string;
    resourceIds: string[];
    tenantId: string;
    actor: string;
    reason?: string;
  }) => Promise<BulkTrashOutcome[]>;
  isTrashing: boolean;
};

export function useBulkTrash(client: LifecycleClient): UseBulkTrashResult {
  const [isTrashing, setIsTrashing] = useState(false);

  const bulkTrash = useCallback(
    async (params: {
      resourceType: string;
      resourceIds: string[];
      tenantId: string;
      actor: string;
      reason?: string;
    }) => {
      setIsTrashing(true);
      const batchId = generateBatchId();
      const outcomes: BulkTrashOutcome[] = [];
      try {
        for (const id of params.resourceIds) {
          try {
            await client.trash({
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
        setIsTrashing(false);
      }
      return outcomes;
    },
    [client],
  );

  return { bulkTrash, isTrashing };
}

function generateBatchId(): string {
  const g = (globalThis as { crypto?: { randomUUID?: () => string } }).crypto;
  if (g?.randomUUID) return g.randomUUID();
  return `batch-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}
