import localforage from "localforage";

const store = localforage.createInstance({ name: "reearth-collab-offline" });

function queueKey(projectId: string): string {
  return `outbox:${projectId}`;
}

export async function collabOfflinePush(
  projectId: string,
  raw: string
): Promise<void> {
  const key = queueKey(projectId);
  const prev = (await store.getItem<string[]>(key)) ?? [];
  prev.push(raw);
  await store.setItem(key, prev);
}

/** Removes and returns queued messages (FIFO). */
export async function collabOfflineDrain(projectId: string): Promise<string[]> {
  const key = queueKey(projectId);
  const prev = (await store.getItem<string[]>(key)) ?? [];
  await store.removeItem(key);
  return prev;
}
