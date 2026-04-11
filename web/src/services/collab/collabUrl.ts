/** Build WebSocket URL for real-time collaboration (matches server GET /api/collab/ws). */
export function buildCollabWsUrl(
  apiBase: string,
  projectId: string,
  accessToken?: string
): string {
  const trimmed = apiBase.replace(/\/$/, "");
  const u = new URL(trimmed);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  u.pathname = `${u.pathname.replace(/\/$/, "")}/collab/ws`;
  u.searchParams.set("projectId", projectId);
  if (accessToken) {
    u.searchParams.set("access_token", accessToken);
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
