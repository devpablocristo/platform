import { useCallback, useState } from "react";
import type { LifecycleClient } from "../api/lifecycleClient";
import type { RestoreRequest } from "../types";

export type UseRestoreResult = {
  restore: (req: RestoreRequest) => Promise<void>;
  isRestoring: boolean;
  error: Error | null;
};

export function useRestore(client: LifecycleClient): UseRestoreResult {
  const [isRestoring, setIsRestoring] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const restore = useCallback(
    async (req: RestoreRequest) => {
      setIsRestoring(true);
      setError(null);
      try {
        await client.restore(req);
      } catch (err) {
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        throw e;
      } finally {
        setIsRestoring(false);
      }
    },
    [client],
  );

  return { restore, isRestoring, error };
}
