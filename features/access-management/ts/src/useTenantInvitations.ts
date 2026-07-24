import { useCallback, useEffect, useRef, useState } from "react";

import type { AccessInvitation, AccessManagementClient } from "./types";

export interface TenantInvitationsState {
  rows: AccessInvitation[];
  loading: boolean;
  error: Error | null;
  reload: () => void;
}

export function useTenantInvitations(
  client: AccessManagementClient,
  tenantId: string | null
): TenantInvitationsState {
  const requestSequence = useRef(0);
  const activeTenantId = useRef(tenantId);
  activeTenantId.current = tenantId;
  const [state, setState] = useState<{
    tenantId: string | null;
    rows: AccessInvitation[];
    loading: boolean;
    error: Error | null;
  }>({ tenantId, rows: [], loading: Boolean(tenantId), error: null });

  const load = useCallback(async () => {
    const requestedTenantId = tenantId;
    const request = ++requestSequence.current;
    setState({ tenantId: requestedTenantId, rows: [], loading: Boolean(requestedTenantId), error: null });
    if (!requestedTenantId) return;

    try {
      const rows = await client.listInvitations(requestedTenantId);
      if (request === requestSequence.current && activeTenantId.current === requestedTenantId) {
        setState({ tenantId: requestedTenantId, rows, loading: false, error: null });
      }
    } catch (error) {
      if (request === requestSequence.current && activeTenantId.current === requestedTenantId) {
        setState({
          tenantId: requestedTenantId,
          rows: [],
          loading: false,
          error: error instanceof Error ? error : new Error(String(error)),
        });
      }
    }
  }, [client, tenantId]);

  useEffect(() => {
    void load();
    return () => {
      requestSequence.current += 1;
    };
  }, [load]);

  const belongsToActiveTenant = state.tenantId === tenantId;
  return {
    rows: belongsToActiveTenant ? state.rows : [],
    loading: state.loading || Boolean(tenantId && !belongsToActiveTenant),
    error: belongsToActiveTenant ? state.error : null,
    reload: load,
  };
}
