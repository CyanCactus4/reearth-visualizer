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

import { CollabClient } from "./CollabClient";
import { CollabContext, type CollabContextValue } from "./collabContext";
import { collabOfflineDrain, collabOfflinePush } from "./offlineQueue";

type Props = {
  projectId?: string;
  children: ReactNode;
};

/**
 * Keeps a WebSocket to /api/collab/ws while the editor has a project id.
 * Does not yet drive UI; safe no-op when projectId is missing.
 */
export const CollabProvider: FC<Props> = ({ projectId, children }) => {
  const { getAccessToken } = useAuth();
  const [status, setStatus] = useState<CollabContextValue["status"]>("idle");
  const [lastMessage, setLastMessage] = useState<CollabContextValue["lastMessage"]>(null);
  const clientRef = useRef<CollabClient | null>(null);

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
          if (msg.t !== "pong") {
            setLastMessage(msg);
          }
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
  }, [projectId, getAccessToken]);

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

  const value = useMemo<CollabContextValue>(
    () => ({
      status,
      projectId,
      lastMessage,
      sendRaw
    }),
    [status, projectId, lastMessage, sendRaw]
  );

  return (
    <CollabContext.Provider value={value}>{children}</CollabContext.Provider>
  );
};
