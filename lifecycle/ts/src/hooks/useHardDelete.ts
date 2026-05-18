import { useCallback, useState } from "react";
import type { ArchiveClient } from "../api/archiveClient";
import type { HardDeleteRequest } from "../types";

export type UseHardDeleteResult = {
  hardDelete: (req: HardDeleteRequest) => Promise<void>;
  isDeleting: boolean;
  error: Error | null;
};

export function useHardDelete(client: ArchiveClient): UseHardDeleteResult {
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const hardDelete = useCallback(
    async (req: HardDeleteRequest) => {
      setIsDeleting(true);
      setError(null);
      try {
        await client.hardDelete(req);
      } catch (err) {
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        throw e;
      } finally {
        setIsDeleting(false);
      }
    },
    [client],
  );

  return { hardDelete, isDeleting, error };
}
