import {
  collabUserAvatarLetter,
  collabUserColor,
  isSameCollabTab,
  parsePeerInstanceKey,
  useCollab
} from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { type CSSProperties, FC, useEffect, useMemo, useRef, useState } from "react";

const panelStyle: CSSProperties = {
  position: "absolute",
  left: 0,
  top: "100%",
  marginTop: 6,
  minWidth: 280,
  maxWidth: 360,
  maxHeight: 340,
  overflowY: "auto",
  zIndex: 50,
  padding: "10px 12px",
  borderRadius: 10,
  background: "rgba(22,24,28,0.97)",
  border: "1px solid rgba(255,255,255,0.12)",
  boxShadow: "0 12px 40px rgba(0,0,0,0.45)"
};

const CollabParticipantsPopover: FC<{ onClose: () => void }> = ({
  onClose
}) => {
  const collab = useCollab();
  const t = useT();
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      const el = rootRef.current;
      if (!el || !(e.target instanceof Node) || el.contains(e.target)) return;
      onClose();
    };
    document.addEventListener("mousedown", onDoc, true);
    return () => document.removeEventListener("mousedown", onDoc, true);
  }, [onClose]);

  const peers = collab?.presencePeerKeys ?? [];
  const typing = new Set(collab?.remoteTypingUserIds ?? []);
  const moving = new Set(collab?.remoteMovingUserIds ?? []);
  const photos = collab?.remoteUserPhotoURLs ?? {};
  const localPhoto = collab?.localUserPhotoURL;
  const localUserId = collab?.localUserId;
  const replica = collab?.collabReplicaId ?? "";

  return (
    <div ref={rootRef} style={panelStyle} role="dialog" aria-label={t("Collab participants title")}>
      <div
        style={{
          fontSize: 12,
          fontWeight: 600,
          marginBottom: 8,
          opacity: 0.95
        }}
      >
        {t("Collab participants heading", { count: peers.length })}
      </div>
      {peers.length === 0 ? (
        <div style={{ fontSize: 12, opacity: 0.75, lineHeight: 1.45 }}>
          {collab?.status === "open"
            ? t("Collab participants empty")
            : t("Collab participants need live")}
        </div>
      ) : (
        <ul style={{ margin: 0, padding: 0, listStyle: "none" }}>
          {peers.map((pk) => {
            const { userId: uid, clientId: cid } = parsePeerInstanceKey(pk);
            const youHere = isSameCollabTab(
              localUserId,
              replica,
              uid,
              cid
            );
            const youElsewhere =
              !!localUserId &&
              uid === localUserId &&
              !youHere;
            let roleLabel: string;
            if (youHere) {
              roleLabel = t("Collab participants you this tab");
            } else if (youElsewhere) {
              roleLabel = t("Collab participants you other tab");
            } else {
              roleLabel = uid.length <= 14 ? uid : `${uid.slice(0, 12)}…`;
            }
            const src =
              photos[uid] || (localUserId === uid ? localPhoto : undefined);
            const canImg =
              src && /^https?:\/\//i.test(String(src).trim());
            const letter = collabUserAvatarLetter(uid);
            const color = collabUserColor(uid);
            const statusBits: string[] = [];
            if (typing.has(pk)) statusBits.push(t("Collab participants status typing"));
            if (moving.has(pk)) statusBits.push(t("Collab participants status moving"));
            return (
              <li
                key={pk}
                style={{
                  display: "flex",
                  alignItems: "flex-start",
                  gap: 10,
                  padding: "8px 0",
                  borderBottom: "1px solid rgba(255,255,255,0.06)"
                }}
              >
                {canImg ? (
                  <img
                    alt=""
                    src={String(src).trim()}
                    referrerPolicy="no-referrer"
                    style={{
                      width: 32,
                      height: 32,
                      borderRadius: 8,
                      objectFit: "cover",
                      flexShrink: 0
                    }}
                  />
                ) : (
                  <span
                    style={{
                      width: 32,
                      height: 32,
                      borderRadius: 8,
                      background: color,
                      color: "#fff",
                      fontSize: 13,
                      fontWeight: 700,
                      lineHeight: "32px",
                      textAlign: "center",
                      flexShrink: 0
                    }}
                  >
                    {letter}
                  </span>
                )}
                <div style={{ minWidth: 0, flex: 1 }}>
                  <div
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      lineHeight: 1.35,
                      wordBreak: "break-all"
                    }}
                  >
                    {roleLabel}
                  </div>
                  {(youHere || youElsewhere) && (
                    <div
                      style={{
                        fontSize: 10,
                        opacity: 0.65,
                        marginTop: 2,
                        wordBreak: "break-all"
                      }}
                    >
                      {uid}
                      {cid ? ` · ${cid.slice(0, 8)}…` : ""}
                    </div>
                  )}
                  {statusBits.length > 0 ? (
                    <div
                      style={{
                        fontSize: 10,
                        opacity: 0.72,
                        marginTop: 4
                      }}
                    >
                      {statusBits.join(" · ")}
                    </div>
                  ) : null}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
};

/** Presence strip + optional participants panel for the collab WebSocket room. */
const CollabPresenceBar: FC = () => {
  const collab = useCollab();
  const t = useT();
  const [panelOpen, setPanelOpen] = useState(false);
  const list = collab?.presencePeerKeys ?? [];

  const typingLine = useMemo(() => {
    const ids = collab?.remoteTypingUserIds ?? [];
    if (ids.length === 0) return null;
    const labels = ids.map((k) => {
      const { userId } = parsePeerInstanceKey(k);
      return userId.length <= 10 ? userId : `${userId.slice(0, 8)}…`;
    });
    return ` · typing: ${labels.join(", ")}`;
  }, [collab?.remoteTypingUserIds]);

  const movingLine = useMemo(() => {
    const ids = collab?.remoteMovingUserIds ?? [];
    if (ids.length === 0) return null;
    const labels = ids.map((k) => {
      const { userId } = parsePeerInstanceKey(k);
      return userId.length <= 10 ? userId : `${userId.slice(0, 8)}…`;
    });
    return ` · moving map: ${labels.join(", ")}`;
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
        position: "relative",
        fontSize: 11,
        lineHeight: 1.4,
        padding: "2px 8px",
        opacity: 0.85,
        borderBottom: "1px solid rgba(255,255,255,0.08)",
        display: "flex",
        alignItems: "center",
        gap: 8,
        flexWrap: "wrap"
      }}
    >
      <span>
        Live: {collab.status}
        {list.length > 0
          ? ` · ${t("Collab presence connections", { count: list.length })}: ${list
              .map((pk) => {
                const { userId, clientId } = parsePeerInstanceKey(pk);
                if (!clientId) {
                  return userId.length <= 12
                    ? userId
                    : `${userId.slice(0, 10)}…`;
                }
                const u =
                  userId.length <= 8 ? userId : `${userId.slice(0, 6)}…`;
                return `${u}·${clientId.slice(0, 4)}`;
              })
              .join(", ")}`
          : null}
        {typingLine}
        {movingLine}
      </span>
      {list.length > 0 ? (
        <span style={{ display: "inline-flex", gap: 3, alignItems: "center" }}>
          {list.map((pk) => {
            const { userId: uid } = parsePeerInstanceKey(pk);
            const src =
              photos[uid] ||
              (collab.localUserId === uid ? localPhoto : undefined);
            if (!src || !/^https?:\/\//i.test(src)) return null;
            return (
              <img
                key={pk}
                src={src}
                alt=""
                title={pk}
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
      <button
        type="button"
        data-testid="collab-participants-toggle"
        onClick={() => setPanelOpen((o) => !o)}
        style={{
          fontSize: 11,
          marginLeft: "auto",
          padding: "4px 10px",
          borderRadius: 6,
          border: "1px solid rgba(255,255,255,0.2)",
          background: "rgba(255,255,255,0.08)",
          color: "inherit",
          cursor: "pointer"
        }}
      >
        {panelOpen
          ? t("Collab participants close")
          : t("Collab participants open")}
      </button>
      {panelOpen ? (
        <CollabParticipantsPopover onClose={() => setPanelOpen(false)} />
      ) : null}
    </div>
  );
};

export default CollabPresenceBar;
