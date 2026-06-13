import { useCallback, useState } from "react";
import type { LifecycleClient } from "../api/lifecycleClient";
import type { PurgeRequest } from "../types";

export type UsePurgeResult = {
  purge: (req: PurgeRequest) => Promise<void>;
  isPurging: boolean;
  error: Error | null;
};

export function usePurge(client: LifecycleClient): UsePurgeResult {
  const [isPurging, setIsPurging] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const purge = useCallback(
    async (req: PurgeRequest) => {
      setIsPurging(true);
      setError(null);
      try {
        await client.purge(req);
      } catch (err) {
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        throw e;
      } finally {
        setIsPurging(false);
      }
    },
    [client],
  );

  return { purge, isPurging, error };
}
