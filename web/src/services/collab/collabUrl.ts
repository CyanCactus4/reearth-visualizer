/** Build WebSocket URL for real-time collaboration (matches server GET /api/collab/ws). */
export function buildCollabWsUrl(
  apiBase: string,
  projectId: string,
  accessToken?: string,
  /** Per-browser-tab id; server scopes collab locks to this connection. */
  clientId?: string
): string {
  const trimmed = apiBase.replace(/\/$/, "");
  const u = new URL(trimmed);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  u.pathname = `${u.pathname.replace(/\/$/, "")}/collab/ws`;
  u.searchParams.set("projectId", projectId);
  if (accessToken) {
    u.searchParams.set("access_token", accessToken);
  }
  const cid = clientId?.trim();
  if (cid) {
    u.searchParams.set("clientId", cid);
  }
  return u.toString();
}

/** HTTP URL for collab chat history (GET /api/collab/chat). */
export function buildCollabChatUrl(
  apiBase: string,
  projectId: string,
  limit?: number
): string {
  const trimmed = apiBase.replace(/\/$/, "");
  const u = new URL(trimmed, window.location.origin);
  u.pathname = `${u.pathname.replace(/\/$/, "")}/collab/chat`;
  u.searchParams.set("projectId", projectId);
  if (limit != null && limit > 0) {
    u.searchParams.set("limit", String(limit));
  }
  return u.toString();
}

/** HTTP URL for collab apply audit (GET /api/collab/apply-audit). */
export function buildCollabApplyAuditUrl(
  apiBase: string,
  projectId: string,
  limit?: number,
  sceneId?: string
): string {
  const trimmed = apiBase.replace(/\/$/, "");
  const u = new URL(trimmed, window.location.origin);
  u.pathname = `${u.pathname.replace(/\/$/, "")}/collab/apply-audit`;
  u.searchParams.set("projectId", projectId);
  if (limit != null && limit > 0) {
    u.searchParams.set("limit", String(limit));
  }
  if (sceneId != null && sceneId.trim() !== "") {
    u.searchParams.set("sceneId", sceneId.trim());
  }
  return u.toString();
}

function collabPostUrl(apiBase: string, pathSuffix: string): string {
  const trimmed = apiBase.replace(/\/$/, "");
  const u = new URL(trimmed, window.location.origin);
  u.pathname = `${u.pathname.replace(/\/$/, "")}${pathSuffix}`;
  return u.toString();
}

/** POST /api/collab/undo */
export function buildCollabUndoPostUrl(apiBase: string): string {
  return collabPostUrl(apiBase, "/collab/undo");
}

/** POST /api/collab/redo */
export function buildCollabRedoPostUrl(apiBase: string): string {
  return collabPostUrl(apiBase, "/collab/redo");
}

export async function postCollabUndo(
  apiBase: string,
  getToken: () => Promise<string | null>,
  sceneId: string,
  /** Same tab id as collab WebSocket — included in fan-out `applied` so other tabs see the action. */
  clientId?: string
): Promise<Response> {
  const token = await getToken();
  const body: Record<string, string> = { sceneId };
  const cid = clientId?.trim();
  if (cid) body.clientId = cid;
  return fetch(buildCollabUndoPostUrl(apiBase), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify(body)
  });
}

export async function postCollabRedo(
  apiBase: string,
  getToken: () => Promise<string | null>,
  sceneId: string,
  clientId?: string
): Promise<Response> {
  const token = await getToken();
  const body: Record<string, string> = { sceneId };
  const cid = clientId?.trim();
  if (cid) body.clientId = cid;
  return fetch(buildCollabRedoPostUrl(apiBase), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify(body)
  });
}

/** POST /api/collab/admin/restore-scene (maintainer+; snapshots must be configured). */
export async function postCollabAdminRestore(
  apiBase: string,
  getToken: () => Promise<string | null>,
  sceneId: string,
  targetSceneRev: number
): Promise<Response> {
  const token = await getToken();
  return fetch(collabPostUrl(apiBase, "/collab/admin/restore-scene"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify({ sceneId, targetSceneRev })
  });
}
