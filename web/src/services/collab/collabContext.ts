import { createContext } from "react";

import type { CollabInbound } from "./CollabClient";

export type CollabStatus = "idle" | "connecting" | "open" | "closed" | "error";

export type RemoteCursor = {
  x: number;
  y: number;
  inside: boolean;
  ts: number;
};

import type { CollabHlcWire } from "./hlc";

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
  /** When set, only that browser tab (see `collabReplicaId`) holds the lock for this user. */
  holderClientId?: string;
  until?: string;
};

export type CollabContextValue = {
  status: CollabStatus;
  projectId: string | undefined;
  /** Current user id (from GraphQL me) — with `collabReplicaId` distinguishes this tab from your other windows. */
  localUserId: string | undefined;
  /** Current user profile photo (GraphQL `me.metadata.photoURL`) for local UI. */
  localUserPhotoURL: string | undefined;
  /** Peer userId → photo URL from WS `presence` join (server passes accounts metadata). */
  remoteUserPhotoURLs: Readonly<Record<string, string>>;
  /**
   * Active collab room members (see `peerInstanceKey`), from `presence_snapshot`
   * plus join/leave deltas. Sorted for stable UI.
   */
  presencePeerKeys: readonly string[];
  lastMessage: CollabInbound | null;
  /** Returns true if the frame was sent on the socket; false if queued offline. */
  sendRaw: (json: string) => boolean;
  /** Keys: `userId` or `userId\u001fclientId` for another tab of the same account. */
  remoteCursors: Record<string, RemoteCursor>;
  /** Peer instance keys (see `remoteCursors`) recently typing outside this tab. */
  remoteTypingUserIds: readonly string[];
  /** Peer instance keys recently panning/zooming the map (`activity` kind `move`). */
  remoteMovingUserIds: readonly string[];
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
  /**
   * Per property-field LWW clock from last `applied.propertyFieldClock` (see `propertyFieldClockKey`).
   */
  propertyFieldClocks: Readonly<Record<string, number>>;
  /** Per property document CAS clock (`propertyDocClockKey`) from `applied.propertyDocClock`. */
  propertyDocClocks: Readonly<Record<string, number>>;
  /** Stable id for this tab (Hybrid Logical Clock replica). */
  collabReplicaId: string;
  /** Next HLC stamp for `update_property_value` CRDT applies (LWW register). */
  tickPropertyFieldHlc: () => CollabHlcWire;
};

export const CollabContext = createContext<CollabContextValue | null>(null);
