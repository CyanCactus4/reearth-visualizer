/** Separates userId and per-tab clientId in composite presence keys. */
export const COLLAB_PEER_SEP = "\u001f";

export function peerInstanceKey(userId: string, clientId?: string): string {
  if (clientId && clientId.length > 0) {
    return `${userId}${COLLAB_PEER_SEP}${clientId}`;
  }
  return userId;
}

export function parsePeerInstanceKey(key: string): {
  userId: string;
  clientId?: string;
} {
  const i = key.indexOf(COLLAB_PEER_SEP);
  if (i < 0) {
    return { userId: key };
  }
  const userId = key.slice(0, i);
  const clientId = key.slice(i + COLLAB_PEER_SEP.length);
  return {
    userId,
    clientId: clientId.length > 0 ? clientId : undefined
  };
}

/** True when peer is this browser tab (same account + same collab client id). */
export function isSameCollabTab(
  localUserId: string | undefined,
  localClientId: string,
  peerUserId: string,
  peerClientId?: string
): boolean {
  if (!localUserId || peerUserId !== localUserId) {
    return false;
  }
  if (!peerClientId) {
    return true;
  }
  return peerClientId === localClientId;
}

/**
 * Skip the "peer applied" toast only when the sender tab matches this tab (same user + same tab key).
 * Uses the same composite key as cursors/presence so it stays consistent with `clientId` on `applied`.
 */
export function suppressCollabPeerAppliedNotification(
  localUserId: string | undefined,
  localClientId: string,
  peerUserId: string,
  peerClientId?: string
): boolean {
  if (!localUserId || peerUserId !== localUserId) {
    return false;
  }
  return (
    peerInstanceKey(peerUserId, peerClientId) ===
    peerInstanceKey(localUserId, localClientId.trim() || undefined)
  );
}
