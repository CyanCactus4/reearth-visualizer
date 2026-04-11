import { useEffect, useMemo } from "react";

import { collabResourceLockKey, lockPayload, type LockResource } from "./lockMessages";
import { useCollab } from "./useCollab";

const HEARTBEAT_MS = 45_000;

/**
 * Acquires a collab lock while `active`, sends heartbeat, releases on cleanup (FR-4).
 */
export function useCollabLockLease(
  resource: LockResource,
  id: string | undefined,
  active: boolean
): void {
  const collab = useCollab();

  useEffect(() => {
    if (!active || !id || collab?.status !== "open" || !collab.projectId) {
      return;
    }
    const send = collab.sendRaw;
    send(lockPayload("acquire", resource, id));
    const hb = window.setInterval(() => {
      send(lockPayload("heartbeat", resource, id));
    }, HEARTBEAT_MS);
    return () => {
      window.clearInterval(hb);
      send(lockPayload("release", resource, id));
    };
  }, [active, id, resource, collab?.status, collab?.projectId, collab?.sendRaw]);
}

export function useForeignCollabLock(
  resource: LockResource,
  id: string | undefined
): {
  readOnly: boolean;
  holderUserId: string | undefined;
} {
  const collab = useCollab();

  return useMemo(() => {
    if (!collab || !id) {
      return { readOnly: false, holderUserId: undefined };
    }
    const key = collabResourceLockKey(resource, id);
    const row = collab.resourceLocks[key];
    if (!row?.holderUserId) {
      return { readOnly: false, holderUserId: undefined };
    }
    const me = collab.localUserId;
    const foreign = !me || row.holderUserId !== me;
    return { readOnly: foreign, holderUserId: row.holderUserId };
  }, [collab, resource, id]);
}
