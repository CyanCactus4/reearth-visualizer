import { createContext } from "react";

import type { CollabInbound } from "./CollabClient";

export type CollabStatus = "idle" | "connecting" | "open" | "closed" | "error";

export type RemoteCursor = {
  x: number;
  y: number;
  inside: boolean;
  ts: number;
};

/** Server-side object lock (layer / widget), from lock_changed / lock_denied. */
export type CollabResourceLock = {
  holderUserId: string;
  until?: string;
};

export type CollabContextValue = {
  status: CollabStatus;
  projectId: string | undefined;
  /** Current user id (from GraphQL me) — used to ignore self in cursor/typing fan-out. */
  localUserId: string | undefined;
  lastMessage: CollabInbound | null;
  sendRaw: (json: string) => void;
  remoteCursors: Record<string, RemoteCursor>;
  /** Peers recently reported typing (via activity); cleared on timeout server-side spacing applies too. */
  remoteTypingUserIds: readonly string[];
  /** Map key `layer:id` / `widget:id` → current holder (TASK.md FR-4). */
  resourceLocks: Readonly<Record<string, CollabResourceLock>>;
};

export const CollabContext = createContext<CollabContextValue | null>(null);
