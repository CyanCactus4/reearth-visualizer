import {
  collabUserAvatarLetter,
  collabUserColor,
  parsePeerInstanceKey,
  useCollab
} from "@reearth/services/collab";
import { FC, useMemo, useState } from "react";

function CollabPeerAvatarChip({
  userId,
  photoUrl
}: {
  userId: string;
  photoUrl?: string;
}) {
  const [imgFailed, setImgFailed] = useState(false);
  const letter = collabUserAvatarLetter(userId);
  const color = collabUserColor(userId);
  const canImg =
    photoUrl &&
    !imgFailed &&
    /^https?:\/\//i.test(photoUrl.trim());
  if (canImg) {
    return (
      <img
        alt=""
        src={photoUrl.trim()}
        referrerPolicy="no-referrer"
        onError={() => setImgFailed(true)}
        style={{
          width: 22,
          height: 22,
          borderRadius: 999,
          objectFit: "cover",
          boxShadow: "0 0 0 2px rgba(255,255,255,0.9), 0 0 0 1px rgba(0,0,0,0.35)"
        }}
      />
    );
  }
  return (
    <span
      aria-label={`Avatar of ${userId}`}
      data-testid={`collab-cursor-avatar-${userId}`}
      style={{
        minWidth: 22,
        height: 22,
        padding: "0 4px",
        borderRadius: 999,
        background: color,
        boxShadow:
          "0 0 0 2px rgba(255,255,255,0.9), 0 0 0 1px rgba(0,0,0,0.35)",
        color: "#fff",
        fontSize: 10,
        fontWeight: 700,
        lineHeight: "22px",
        textAlign: "center",
        letterSpacing: -0.3
      }}
    >
      {letter}
    </span>
  );
}

function shortUserLabel(userId: string): string {
  return userId.length <= 10 ? userId : `${userId.slice(0, 8)}…`;
}

/** Remote collaborator cursors over the visualizer (normalized coords from server). */
const CollabRemoteCursors: FC = () => {
  const collab = useCollab();
  const entries = useMemo(() => {
    if (!collab?.remoteCursors) return [];
    return Object.entries(collab.remoteCursors).filter(
      ([, c]) => c.inside
    );
  }, [collab?.remoteCursors]);

  if (!collab?.projectId || entries.length === 0) {
    return null;
  }

  return (
    <div
      data-testid="collab-remote-cursors"
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
        zIndex: 30,
        overflow: "hidden"
      }}
    >
      {entries.map(([peerKey, c]) => {
        const { userId } = parsePeerInstanceKey(peerKey);
        return (
        <div
          key={peerKey}
          title={userId}
          style={{
            position: "absolute",
            left: `${c.x * 100}%`,
            top: `${c.y * 100}%`,
            transform: "translate(-4px,-4px)",
            display: "flex",
            alignItems: "center",
            gap: 4,
            whiteSpace: "nowrap"
          }}
        >
          <CollabPeerAvatarChip
            userId={userId}
            photoUrl={collab?.remoteUserPhotoURLs[userId]}
          />
          <span
            style={{
              fontSize: 10,
              lineHeight: 1.2,
              padding: "1px 5px",
              borderRadius: 4,
              background: "rgba(0,0,0,0.55)",
              color: "#fff"
            }}
          >
            {shortUserLabel(userId)}
          </span>
        </div>
        );
      })}
    </div>
  );
};

export default CollabRemoteCursors;
