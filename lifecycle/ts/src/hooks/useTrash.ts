import { useCallback, useState } from "react";
import type { LifecycleClient } from "../api/lifecycleClient";
import type { TrashRequest } from "../types";

export type UseTrashResult = {
  trash: (req: TrashRequest) => Promise<void>;
  isTrashing: boolean;
  error: Error | null;
};

export function useTrash(client: LifecycleClient): UseTrashResult {
  const [isTrashing, setIsTrashing] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const trash = useCallback(
    async (req: TrashRequest) => {
      setIsTrashing(true);
      setError(null);
      try {
        await client.trash(req);
      } catch (err) {
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        throw e;
      } finally {
        setIsTrashing(false);
      }
    },
    [client],
  );

  return { trash, isTrashing, error };
}
