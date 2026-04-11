import { collabUserColor, useCollab } from "@reearth/services/collab";
import { FC, useMemo } from "react";

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
      {entries.map(([userId, c]) => (
        <div
          key={userId}
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
          <span
            style={{
              width: 10,
              height: 10,
              borderRadius: "50%",
              background: collabUserColor(userId),
              boxShadow: "0 0 0 1px rgba(0,0,0,0.35)"
            }}
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
      ))}
    </div>
  );
};

export default CollabRemoteCursors;
