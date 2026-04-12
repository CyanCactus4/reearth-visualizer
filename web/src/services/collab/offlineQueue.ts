import localforage from "localforage";

const store = localforage.createInstance({ name: "reearth-collab-offline" });

function queueKey(projectId: string): string {
  return `outbox:${projectId}`;
}

export type OfflineCollabEntry = {
  id: string;
  raw: string;
  ts: number;
};

function isLegacyStringQueue(v: unknown): v is string[] {
  return (
    Array.isArray(v) &&
    v.length > 0 &&
    typeof (v as unknown[])[0] === "string"
  );
}

/** Normalizes legacy string[] queues to OfflineCollabEntry[]. */
export async function collabOfflineNormalize(
  projectId: string
): Promise<OfflineCollabEntry[]> {
  const key = queueKey(projectId);
  const v = await store.getItem<unknown>(key);
  if (!v) return [];
  if (isLegacyStringQueue(v)) {
    const migrated: OfflineCollabEntry[] = v.map((raw) => ({
      id: crypto.randomUUID(),
      raw,
      ts: Date.now()
    }));
    await store.setItem(key, migrated);
    return migrated;
  }
  return (v as OfflineCollabEntry[]) ?? [];
}

export async function collabOfflinePush(
  projectId: string,
  raw: string
): Promise<void> {
  const key = queueKey(projectId);
  const prev = await collabOfflineNormalize(projectId);
  prev.push({ id: crypto.randomUUID(), raw, ts: Date.now() });
  await store.setItem(key, prev);
}

/** Removes and returns queued messages (FIFO) — legacy API; prefer collabOfflineFlush. */
export async function collabOfflineDrain(projectId: string): Promise<string[]> {
  const key = queueKey(projectId);
  const prev = await collabOfflineNormalize(projectId);
  await store.removeItem(key);
  return prev.map((e) => e.raw);
}

/**
 * Sends queued frames in order until the socket refuses; keeps unsent tail so nothing is lost
 * if the connection drops mid-flush (TASK offline / graceful degradation).
 */
export async function collabOfflineFlush(
  projectId: string,
  trySend: (raw: string) => boolean
): Promise<void> {
  const key = queueKey(projectId);
  let entries = await collabOfflineNormalize(projectId);
  if (!entries.length) return;
  let i = 0;
  for (; i < entries.length; i++) {
    if (!trySend(entries[i].raw)) {
      break;
    }
  }
  if (i >= entries.length) {
    await store.removeItem(key);
  } else {
    await store.setItem(key, entries.slice(i));
  }
}
