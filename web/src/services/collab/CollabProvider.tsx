import { useAuth } from "@reearth/services/auth/useAuth";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
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
import { fetchCollabChatHistory } from "./collabChatApi";
import { CollabClient, type CollabInbound } from "./CollabClient";
import { chatPayload } from "./chatMessages";
import {
  CollabContext,
  type CollabChatLine,
  type CollabContextValue,
  type CollabResourceLock,
  type RemoteCursor
} from "./collabContext";
import CollabLockConflictModal, {
  type CollabLockConflictPayload,
  type CollabLockConflictSnapshots
} from "./CollabLockConflictModal";
import {
  collabResourceLockKey,
  type LockResource
} from "./lockMessages";
import { extractChatMentions } from "./chatMentions";
import { collabOfflineDrain, collabOfflinePush } from "./offlineQueue";

const lockResourceKinds = new Set<string>([
  "layer",
  "widget",
  "scene",
  "widget_area"
]);

function asLockResource(s: string): LockResource | null {
  return lockResourceKinds.has(s) ? (s as LockResource) : null;
}

type Props = {
  projectId?: string;
  /** GraphQL `me.id` — omit when unknown; own cursor/typing events are ignored. */
  localUserId?: string;
  /** Refetch scene from server (e.g. user chose “reload” after lock conflict). */
  onReconcileScene?: () => void;
  /** Optional: load two lightweight scene snapshots for merge-compare UI. */
  onLockConflictCompare?: () => Promise<CollabLockConflictSnapshots | null>;
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
  onReconcileScene,
  onLockConflictCompare,
  children
}) => {
  const { getAccessToken } = useAuth();
  const getAccessTokenRef = useRef(getAccessToken);
  useEffect(() => {
    getAccessTokenRef.current = getAccessToken;
  }, [getAccessToken]);
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
  const [chatMessages, setChatMessages] = useState<CollabChatLine[]>([]);
  const [remoteSceneRev, setRemoteSceneRev] = useState<number | undefined>(
    undefined
  );
  const seenChatIdsRef = useRef<Set<string>>(new Set());
  /** Maps userId+NUL+text → optimistic local id (one in-flight own line per text). */
  const optimisticByKeyRef = useRef<Map<string, string>>(new Map());
  const clientRef = useRef<CollabClient | null>(null);
  const localUserIdRef = useRef(localUserId);
  const typingTimersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map()
  );
  const lastLocalTypingSent = useRef(0);
  const [, setNotification] = useNotification();
  const tCollab = useT();
  const lastLockDeniedKeyRef = useRef<string | null>(null);
  const lastAppliedNotifyAtRef = useRef<Map<string, number>>(new Map());
  const lastApplyErrorToastAtRef = useRef<Map<string, number>>(new Map());
  const [lockConflict, setLockConflict] =
    useState<CollabLockConflictPayload | null>(null);

  useEffect(() => {
    localUserIdRef.current = localUserId;
  }, [localUserId]);

  useEffect(() => {
    if (lastMessage?.t !== "lock_denied") return;
    const d = lastMessage.d as
      | {
          resource?: string;
          id?: string;
          holderUserId?: string;
        }
      | undefined;
    if (!d?.resource || !d.id || !d.holderUserId) return;
    const k = `${d.resource}:${d.id}:${d.holderUserId}`;
    if (lastLockDeniedKeyRef.current === k) return;
    lastLockDeniedKeyRef.current = k;
    setNotification({
      type: "warning",
      text: tCollab("Collab lock denied toast", {
        userId: d.holderUserId,
        resource: d.resource,
        id: d.id
      })
    });
    setLockConflict({
      resource: d.resource,
      id: d.id,
      holderUserId: d.holderUserId
    });
  }, [lastMessage, setNotification, tCollab]);

  useEffect(() => {
    if (lastMessage?.t !== "applied") return;
    const d = lastMessage.d as
      | { userId?: string; kind?: string; widgetId?: string }
      | undefined;
    const peer = d?.userId;
    if (!peer || peer === "unknown") return;
    if (localUserId && peer === localUserId) return;
    const kind = typeof d?.kind === "string" ? d.kind : "";
    const wid = typeof d?.widgetId === "string" ? d.widgetId : "";
    const key = `${peer}\0${kind}\0${wid}`;
    const now = Date.now();
    const prev = lastAppliedNotifyAtRef.current.get(key) ?? 0;
    if (now - prev < 4000) return;
    lastAppliedNotifyAtRef.current.set(key, now);
    setNotification({
      type: "info",
      text: tCollab("Collab peer applied toast", {
        userId: peer,
        kind: kind || "edit",
        widgetId: wid || "—"
      })
    });
  }, [lastMessage, localUserId, setNotification, tCollab]);

  useEffect(() => {
    if (lastMessage?.t !== "error") return;
    const d = lastMessage.d as
      | { code?: string; message?: string }
      | undefined;
    const code = typeof d?.code === "string" ? d.code : "";
    if (
      code !== "apply_failed" &&
      code !== "object_locked" &&
      code !== "stale_state"
    )
      return;
    const now = Date.now();
    const prev = lastApplyErrorToastAtRef.current.get(code) ?? 0;
    if (now - prev < 3500) return;
    lastApplyErrorToastAtRef.current.set(code, now);
    if (code === "object_locked") {
      setNotification({
        type: "warning",
        text: tCollab("Collab apply object locked toast")
      });
      return;
    }
    if (code === "stale_state") {
      setNotification({
        type: "warning",
        text: tCollab("Collab apply stale toast")
      });
      return;
    }
    setNotification({
      type: "error",
      text: tCollab("Collab apply failed toast", {
        message: typeof d?.message === "string" ? d.message : ""
      })
    });
  }, [lastMessage, setNotification, tCollab]);

  useEffect(() => {
    if (lastMessage?.t !== "notify") return;
    const d = lastMessage.d as
      | { kind?: string; fromUserId?: string; text?: string }
      | undefined;
    if (d?.kind !== "chat_mention") return;
    const from = typeof d.fromUserId === "string" ? d.fromUserId : "";
    setNotification({
      type: "info",
      text: tCollab("Collab chat mention notify", {
        userId: from || "—",
        preview:
          typeof d.text === "string"
            ? d.text.slice(0, 120)
            : ""
      })
    });
  }, [lastMessage, localUserId, setNotification, tCollab]);

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
        const res = asLockResource(d.resource);
        if (!res) return;
        const key = collabResourceLockKey(res, d.id);
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
        const resDenied = asLockResource(d.resource);
        if (!resDenied) return;
        const key = collabResourceLockKey(resDenied, d.id);
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
        return;
      }
      if (msg.t === "chat") {
        const d = msg.d as
          | {
              id?: string;
              userId?: string;
              text?: string;
              ts?: number;
              mentions?: string[];
            }
          | undefined;
        if (!d?.userId || d.text == null || d.text === "") return;
        const mentions =
          Array.isArray(d.mentions) && d.mentions.length > 0
            ? d.mentions.filter((x): x is string => typeof x === "string" && x !== "")
            : undefined;
        const cid =
          d.id && d.id.length > 0
            ? d.id
            : `${d.userId}:${d.ts ?? 0}:${d.text}`;
        const self = localUserIdRef.current;
        if (self && d.userId === self) {
          const fp = `${self}\0${d.text}`;
          const optId = optimisticByKeyRef.current.get(fp);
          if (optId) {
            optimisticByKeyRef.current.delete(fp);
            seenChatIdsRef.current.delete(optId);
            if (seenChatIdsRef.current.has(cid)) {
              setChatMessages((prev) => prev.filter((m) => m.id !== optId));
              return;
            }
            seenChatIdsRef.current.add(cid);
            setChatMessages((prev) =>
              prev.map((m) =>
                m.id === optId
                  ? {
                      id: cid,
                      userId: d.userId!,
                      text: d.text!,
                      ts: d.ts ?? Math.floor(Date.now() / 1000),
                      mentions,
                      pending: false
                    }
                  : m
              )
            );
            return;
          }
        }
        if (seenChatIdsRef.current.has(cid)) return;
        seenChatIdsRef.current.add(cid);
        setChatMessages((prev) => {
          const line: CollabChatLine = {
            id: cid,
            userId: d.userId!,
            text: d.text!,
            ts: d.ts ?? Math.floor(Date.now() / 1000),
            mentions
          };
          const next = [...prev, line];
          next.sort((a, b) => a.ts - b.ts);
          return next;
        });
        return;
      }
      if (msg.t === "applied") {
        const d = msg.d as { sceneRev?: number } | undefined;
        if (typeof d?.sceneRev === "number") {
          setRemoteSceneRev(d.sceneRev);
        }
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
    setChatMessages([]);
    setRemoteSceneRev(undefined);
    seenChatIdsRef.current.clear();
    optimisticByKeyRef.current.clear();
    lastAppliedNotifyAtRef.current.clear();

    if (!projectId) return;

    let cancelled = false;
    const apiBase = window.REEARTH_CONFIG?.api || "/api";
    void (async () => {
      const rows = await fetchCollabChatHistory(
        apiBase,
        projectId,
        () => getAccessTokenRef.current(),
        200
      );
      if (cancelled) return;
      const sorted = rows.slice().sort((a, b) => a.ts - b.ts);
      for (const r of sorted) {
        seenChatIdsRef.current.add(r.id);
      }
      setChatMessages(sorted);
    })();

    return () => {
      cancelled = true;
    };
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
    (json: string): boolean => {
      if (!projectId) return false;
      const sent = clientRef.current?.sendRaw(json) ?? false;
      if (!sent) {
        void collabOfflinePush(projectId, json);
      }
      return sent;
    },
    [projectId]
  );

  const sendChat = useCallback(
    (text: string) => {
      const t = text.trim();
      if (!t) return;
      const uid = localUserIdRef.current;
      if (uid) {
        const fp = `${uid}\0${t}`;
        const prevOpt = optimisticByKeyRef.current.get(fp);
        if (prevOpt) {
          seenChatIdsRef.current.delete(prevOpt);
          optimisticByKeyRef.current.delete(fp);
          setChatMessages((arr) => arr.filter((m) => m.id !== prevOpt));
        }
        const optId = `local:${crypto.randomUUID()}`;
        optimisticByKeyRef.current.set(fp, optId);
        seenChatIdsRef.current.add(optId);
        const ts = Math.floor(Date.now() / 1000);
        const ment = extractChatMentions(t, 20);
        setChatMessages((prev) => {
          const next = [
            ...prev,
            {
              id: optId,
              userId: uid,
              text: t,
              ts,
              mentions: ment.length ? ment : undefined,
              pending: true
            }
          ];
          next.sort((a, b) => a.ts - b.ts);
          return next;
        });
      }
      sendRaw(chatPayload(t));
    },
    [sendRaw]
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
      resourceLocks,
      chatMessages,
      sendChat,
      remoteSceneRev
    }),
    [
      status,
      projectId,
      localUserId,
      lastMessage,
      sendRaw,
      remoteCursors,
      remoteTypingUserIds,
      resourceLocks,
      chatMessages,
      sendChat,
      remoteSceneRev
    ]
  );

  const closeLockConflict = useCallback(() => {
    setLockConflict(null);
    lastLockDeniedKeyRef.current = null;
  }, []);

  return (
    <>
      <CollabContext.Provider value={value}>{children}</CollabContext.Provider>
      <CollabLockConflictModal
        open={!!lockConflict}
        payload={lockConflict}
        onClose={closeLockConflict}
        onReconcileScene={onReconcileScene}
        onCompareSnapshots={onLockConflictCompare}
      />
    </>
  );
};
