import { useCallback, useState } from "react";
import type { LifecycleClient } from "../api/lifecycleClient";
import type { ArchiveRequest } from "../types";

export type UseArchiveResult = {
  archive: (req: ArchiveRequest) => Promise<void>;
  isArchiving: boolean;
  error: Error | null;
};

/**
 * useArchive returns a stable `archive(req)` function plus loading and error
 * state. Designed to be called from a CrudPage action or a confirm dialog.
 *
 * The hook is intentionally minimal: the consumer decides when to refresh
 * data (typically inside its react-query mutation onSuccess). For
 * fire-and-forget archives use the client directly.
 */
export function useArchive(client: LifecycleClient): UseArchiveResult {
  const [isArchiving, setIsArchiving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const archive = useCallback(
    async (req: ArchiveRequest) => {
      setIsArchiving(true);
      setError(null);
      try {
        await client.archive(req);
      } catch (err) {
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        throw e;
      } finally {
        setIsArchiving(false);
      }
    },
    [client],
  );

  return { archive, isArchiving, error };
}
