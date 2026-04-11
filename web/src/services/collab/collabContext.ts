import { createContext } from "react";

import type { CollabInbound } from "./CollabClient";

export type CollabStatus = "idle" | "connecting" | "open" | "closed" | "error";

export type RemoteCursor = {
  x: number;
  y: number;
  inside: boolean;
  ts: number;
};

export type CollabChatLine = {
  id: string;
  userId: string;
  text: string;
  ts: number;
  /** Parsed @handles (no @), from server or local mirror of server rules. */
  mentions?: readonly string[];
  /** True until the server echo replaces this line with a real id. */
  pending?: boolean;
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
  /** Returns true if the frame was sent on the socket; false if queued offline. */
  sendRaw: (json: string) => boolean;
  remoteCursors: Record<string, RemoteCursor>;
  /** Peers recently reported typing (via activity); cleared on timeout server-side spacing applies too. */
  remoteTypingUserIds: readonly string[];
  /** Map key `layer:id` / `widget:id` → current holder (TASK.md FR-4). */
  resourceLocks: Readonly<Record<string, CollabResourceLock>>;
  /** Recent collab chat lines (REST history + live WebSocket). */
  chatMessages: readonly CollabChatLine[];
  sendChat: (text: string) => void;
  /** Monotonic scene stamp from last `applied` collab message (server `scene.UpdatedAt` ms). */
  remoteSceneRev: number | undefined;
  /** Per-widget field LWW clocks from last peer `applied` (server `entityClocks`); used for CRDT-style applies. */
  widgetEntityClocks: Readonly<
    Record<string, Readonly<Record<string, number>>>
  >;
};

export const CollabContext = createContext<CollabContextValue | null>(null);
