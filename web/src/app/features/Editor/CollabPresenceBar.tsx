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

  const movingLine = useMemo(() => {
    const ids = collab?.remoteMovingUserIds ?? [];
    if (ids.length === 0) return null;
    return ` · moving map: ${ids.join(", ")}`;
  }, [collab?.remoteMovingUserIds]);

  const photos = collab?.remoteUserPhotoURLs ?? {};
  const localPhoto = collab?.localUserPhotoURL;

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
        borderBottom: "1px solid rgba(255,255,255,0.08)",
        display: "flex",
        alignItems: "center",
        gap: 6,
        flexWrap: "wrap"
      }}
    >
      <span>
        Live: {collab.status}
        {list.length > 0
          ? ` · ${list.length} active: ${list.join(", ")}`
          : null}
        {typingLine}
        {movingLine}
      </span>
      {list.length > 0 ? (
        <span style={{ display: "inline-flex", gap: 3, alignItems: "center" }}>
          {list.map((uid) => {
            const src =
              photos[uid] ||
              (collab.localUserId === uid ? localPhoto : undefined);
            if (!src || !/^https?:\/\//i.test(src)) return null;
            return (
              <img
                key={uid}
                src={src}
                alt=""
                title={uid}
                referrerPolicy="no-referrer"
                style={{
                  width: 16,
                  height: 16,
                  borderRadius: 4,
                  objectFit: "cover"
                }}
              />
            );
          })}
        </span>
      ) : null}
    </div>
  );
};

export default CollabPresenceBar;
