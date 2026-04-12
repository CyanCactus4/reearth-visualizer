import type { CollabChatLine } from "./collabContext";
import { buildCollabChatUrl } from "./collabUrl";

export async function fetchCollabChatHistory(
  apiBase: string,
  projectId: string,
  getAccessToken: () => Promise<string>,
  limit = 200
): Promise<CollabChatLine[]> {
  try {
    const token = await getAccessToken();
    const url = buildCollabChatUrl(apiBase, projectId, limit);
    const res = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {}
    });
    if (!res.ok) {
      return [];
    }
    const body = (await res.json()) as {
      v?: number;
      messages?: CollabChatLine[];
    };
    if (!Array.isArray(body.messages)) {
      return [];
    }
    return body.messages;
  } catch {
    return [];
  }
}
