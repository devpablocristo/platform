import { useCallback, useState } from "react";
import type { LifecycleClient } from "../api/lifecycleClient";
import type { UnarchiveRequest } from "../types";

export type UseUnarchiveResult = {
  unarchive: (req: UnarchiveRequest) => Promise<void>;
  isUnarchiving: boolean;
  error: Error | null;
};

export function useUnarchive(client: LifecycleClient): UseUnarchiveResult {
  const [isUnarchiving, setIsUnarchiving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const unarchive = useCallback(
    async (req: UnarchiveRequest) => {
      setIsUnarchiving(true);
      setError(null);
      try {
        await client.unarchive(req);
      } catch (err) {
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        throw e;
      } finally {
        setIsUnarchiving(false);
      }
    },
    [client],
  );

  return { unarchive, isUnarchiving, error };
}
