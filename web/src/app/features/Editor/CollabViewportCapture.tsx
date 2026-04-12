import { cursorPayload, useCollab } from "@reearth/services/collab";
import { FC, ReactNode, useCallback, useRef } from "react";

import CollabRemoteCursors from "./CollabRemoteCursors";

const CLIENT_CURSOR_MS = 45;

type Props = { children: ReactNode };

/**
 * Tracks pointer position over the visualizer and sends throttled cursor messages (TASK.md FR-3).
 */
const CollabViewportCapture: FC<Props> = ({ children }) => {
  const collab = useCollab();
  const lastSent = useRef(0);

  const onMouseMove = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (collab?.status !== "open") return;
      const now = Date.now();
      if (now - lastSent.current < CLIENT_CURSOR_MS) return;
      lastSent.current = now;
      const el = e.currentTarget;
      const r = el.getBoundingClientRect();
      if (r.width < 1 || r.height < 1) return;
      const x = Math.min(1, Math.max(0, (e.clientX - r.left) / r.width));
      const y = Math.min(1, Math.max(0, (e.clientY - r.top) / r.height));
      collab.sendRaw(cursorPayload(x, y, true, collab.collabReplicaId));
    },
    [collab]
  );

  const onMouseLeave = useCallback(() => {
    if (collab?.status !== "open") return;
    collab.sendRaw(cursorPayload(0, 0, false, collab.collabReplicaId));
  }, [collab]);

  return (
    <div
      style={{ position: "relative", width: "100%", height: "100%" }}
      onMouseMove={onMouseMove}
      onMouseLeave={onMouseLeave}
    >
      {children}
      <CollabRemoteCursors />
    </div>
  );
};

export default CollabViewportCapture;
