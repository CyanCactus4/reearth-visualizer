import { useCollab } from "@reearth/services/collab";
import { FC, useEffect, useMemo, useState } from "react";

/** Lightweight presence strip for the collab WebSocket (join/leave events). */
const CollabPresenceBar: FC = () => {
  const collab = useCollab();
  const [userIds, setUserIds] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    const m = collab?.lastMessage;
    if (!m || m.t !== "presence") return;
    const d = m.d as { event?: string; userId?: string } | undefined;
    if (!d) return;
    const ev = d.event;
    const uid = d.userId;
    if (!uid) return;
    setUserIds((prev) => {
      const next = new Set(prev);
      if (ev === "join") next.add(uid);
      if (ev === "leave") next.delete(uid);
      return next;
    });
  }, [collab?.lastMessage]);

  const list = useMemo(() => Array.from(userIds).sort(), [userIds]);

  const typingLine = useMemo(() => {
    const ids = collab?.remoteTypingUserIds ?? [];
    if (ids.length === 0) return null;
    return ` · typing: ${ids.join(", ")}`;
  }, [collab?.remoteTypingUserIds]);

  if (!collab?.projectId) {
    return null;
  }

  return (
    <div
      data-testid="collab-presence-bar"
      style={{
        fontSize: 11,
        lineHeight: 1.4,
        padding: "2px 8px",
        opacity: 0.85,
        borderBottom: "1px solid rgba(255,255,255,0.08)"
      }}
    >
      Live: {collab.status}
      {list.length > 0 ? ` · ${list.length} active: ${list.join(", ")}` : null}
      {typingLine}
    </div>
  );
};

export default CollabPresenceBar;
