import { useAuth } from "@reearth/services/auth/useAuth";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type FC,
  type ReactNode
} from "react";

import { activityPayload } from "./activityMessages";
import { CollabClient, type CollabInbound } from "./CollabClient";
import {
  CollabContext,
  type CollabContextValue,
  type CollabResourceLock,
  type RemoteCursor
} from "./collabContext";
import { collabResourceLockKey } from "./lockMessages";
import { collabOfflineDrain, collabOfflinePush } from "./offlineQueue";

type Props = {
  projectId?: string;
  /** GraphQL `me.id` — omit when unknown; own cursor/typing events are ignored. */
  localUserId?: string;
  children: ReactNode;
};

const CURSOR_STALE_MS = 5000;
const TYPING_TTL_MS = 4000;
const LOCAL_TYPING_DEBOUNCE_MS = 2500;

/**
 * Keeps a WebSocket to /api/collab/ws while the editor has a project id.
 * Merges cursor / activity into context for presence UI (TASK.md FR-3).
 */
export const CollabProvider: FC<Props> = ({
  projectId,
  localUserId,
  children
}) => {
  const { getAccessToken } = useAuth();
  const [status, setStatus] = useState<CollabContextValue["status"]>("idle");
  const [lastMessage, setLastMessage] =
    useState<CollabContextValue["lastMessage"]>(null);
  const [remoteCursors, setRemoteCursors] = useState<
    Record<string, RemoteCursor>
  >({});
  const [remoteTypingUserIds, setRemoteTypingUserIds] = useState<string[]>([]);
  const [resourceLocks, setResourceLocks] = useState<
    Record<string, CollabResourceLock>
  >({});
  const clientRef = useRef<CollabClient | null>(null);
  const localUserIdRef = useRef(localUserId);
  const typingTimersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map()
  );
  const lastLocalTypingSent = useRef(0);

  useEffect(() => {
    localUserIdRef.current = localUserId;
  }, [localUserId]);

  const removeTypingUser = useCallback((uid: string) => {
    typingTimersRef.current.delete(uid);
    setRemoteTypingUserIds((arr) => arr.filter((x) => x !== uid));
  }, []);

  const noteTypingUser = useCallback(
    (uid: string) => {
      const self = localUserIdRef.current;
      if (!uid || (self && uid === self)) return;
      const prev = typingTimersRef.current.get(uid);
      if (prev) clearTimeout(prev);
      const t = setTimeout(() => removeTypingUser(uid), TYPING_TTL_MS);
      typingTimersRef.current.set(uid, t);
      setRemoteTypingUserIds((arr) =>
        arr.includes(uid) ? arr : [...arr, uid]
      );
    },
    [removeTypingUser]
  );

  const applyInbound = useCallback(
    (msg: CollabInbound) => {
      if (msg.t === "lock_changed") {
        const d = msg.d as
          | {
              released?: boolean;
              resource?: string;
              id?: string;
              holderUserId?: string;
              until?: string;
            }
          | undefined;
        if (!d?.resource || !d.id) return;
        if (d.resource !== "layer" && d.resource !== "widget") return;
        const key = collabResourceLockKey(d.resource, d.id);
        if (d.released) {
          setResourceLocks((prev) => {
            if (!(key in prev)) return prev;
            const rest: Record<string, CollabResourceLock> = {};
            for (const [k, v] of Object.entries(prev)) {
              if (k !== key) rest[k] = v;
            }
            return rest;
          });
          return;
        }
        const holder = d.holderUserId;
        if (holder) {
          setResourceLocks((prev) => ({
            ...prev,
            [key]: { holderUserId: holder, until: d.until }
          }));
        }
        return;
      }
      if (msg.t === "lock_denied") {
        const d = msg.d as
          | {
              resource?: string;
              id?: string;
              holderUserId?: string;
              until?: string;
            }
          | undefined;
        if (!d?.resource || !d.id || !d.holderUserId) return;
        if (d.resource !== "layer" && d.resource !== "widget") return;
        const key = collabResourceLockKey(d.resource, d.id);
        const holderDenied = d.holderUserId;
        setResourceLocks((prev) => ({
          ...prev,
          [key]: { holderUserId: holderDenied, until: d.until }
        }));
        return;
      }
      if (msg.t === "cursor") {
        const d = msg.d as
          | {
              userId?: string;
              x?: number;
              y?: number;
              inside?: boolean;
            }
          | undefined;
        if (!d || typeof d.x !== "number" || typeof d.y !== "number") return;
        const x = d.x;
        const y = d.y;
        const uid = d.userId;
        const self = localUserIdRef.current;
        if (!uid || (self && uid === self)) return;
        const inside = d.inside !== false;
        setRemoteCursors((prev) => ({
          ...prev,
          [uid]: { x, y, inside, ts: Date.now() }
        }));
        return;
      }
      if (msg.t === "activity") {
        const d = msg.d as
          | { userId?: string; kind?: string }
          | undefined;
        if (!d || d.kind !== "typing" || !d.userId) return;
        noteTypingUser(d.userId);
      }
    },
    [noteTypingUser]
  );

  useEffect(() => {
    setRemoteCursors({});
    setResourceLocks({});
    for (const t of typingTimersRef.current.values()) clearTimeout(t);
    typingTimersRef.current.clear();
    setRemoteTypingUserIds([]);
  }, [projectId]);

  useEffect(() => {
    if (status === "closed" || status === "error") {
      setResourceLocks({});
    }
  }, [status]);

  useEffect(() => {
    if (!projectId) {
      setStatus("idle");
      setLastMessage(null);
      return;
    }

    let cancelled = false;
    const apiBase = window.REEARTH_CONFIG?.api || "/api";
    const client = new CollabClient(apiBase, getAccessToken);
    clientRef.current = client;

    const run = async () => {
      setStatus("connecting");
      try {
        const ws = await client.connect(projectId);
        if (cancelled) {
          ws.close();
          return;
        }
        client.onMessage((msg) => {
          if (msg.t === "pong") return;
          setLastMessage(msg);
          applyInbound(msg);
        });
        ws.onopen = () => {
          setStatus("open");
          client.ping();
        };
        ws.onerror = () => {
          if (!cancelled) setStatus("error");
        };
        ws.onclose = () => {
          if (!cancelled) setStatus("closed");
        };
      } catch {
        if (!cancelled) setStatus("error");
      }
    };

    void run();

    return () => {
      cancelled = true;
      client.disconnect();
      clientRef.current = null;
    };
  }, [projectId, getAccessToken, applyInbound]);

  useEffect(() => {
    if (status !== "open") return;
    const id = window.setInterval(() => {
      const now = Date.now();
      setRemoteCursors((prev) => {
        let changed = false;
        const next: Record<string, RemoteCursor> = {};
        for (const [k, v] of Object.entries(prev)) {
          if (now - v.ts <= CURSOR_STALE_MS) {
            next[k] = v;
          } else {
            changed = true;
          }
        }
        return changed ? next : prev;
      });
    }, 2000);
    return () => clearInterval(id);
  }, [status]);

  useEffect(() => {
    if (status !== "open" || !projectId) return;
    void (async () => {
      const batch = await collabOfflineDrain(projectId);
      for (const line of batch) {
        clientRef.current?.sendRaw(line);
      }
    })();
  }, [status, projectId]);

  useEffect(() => {
    const onOnline = () => {
      if (status !== "open" || !projectId) return;
      void (async () => {
        const batch = await collabOfflineDrain(projectId);
        for (const line of batch) {
          clientRef.current?.sendRaw(line);
        }
      })();
    };
    window.addEventListener("online", onOnline);
    return () => window.removeEventListener("online", onOnline);
  }, [status, projectId]);

  const sendRaw = useCallback(
    (json: string) => {
      if (!projectId) return;
      const sent = clientRef.current?.sendRaw(json) ?? false;
      if (!sent) {
        void collabOfflinePush(projectId, json);
      }
    },
    [projectId]
  );

  useEffect(() => {
    if (status !== "open" || !projectId) return;
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement | null;
      if (!t) return;
      const tag = t.tagName;
      if (
        tag !== "INPUT" &&
        tag !== "TEXTAREA" &&
        !t.isContentEditable
      ) {
        return;
      }
      const now = Date.now();
      if (now - lastLocalTypingSent.current < LOCAL_TYPING_DEBOUNCE_MS) return;
      lastLocalTypingSent.current = now;
      sendRaw(activityPayload("typing"));
    };
    document.addEventListener("keydown", onKey, true);
    return () => document.removeEventListener("keydown", onKey, true);
  }, [status, projectId, sendRaw]);

  const value = useMemo<CollabContextValue>(
    () => ({
      status,
      projectId,
      localUserId,
      lastMessage,
      sendRaw,
      remoteCursors,
      remoteTypingUserIds,
      resourceLocks
    }),
    [
      status,
      projectId,
      localUserId,
      lastMessage,
      sendRaw,
      remoteCursors,
      remoteTypingUserIds,
      resourceLocks
    ]
  );

  return (
    <CollabContext.Provider value={value}>{children}</CollabContext.Provider>
  );
};
