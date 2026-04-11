import { useT } from "@reearth/services/i18n/hooks";
import { FC, ReactNode } from "react";

import type { LockResource } from "./lockMessages";
import { useForeignCollabLock } from "./useCollabResourceLock";

type Props = {
  resource: LockResource;
  id: string | undefined;
  children: ReactNode;
};

/** Dims inspector content when another user holds the collab lock (no lease). */
const CollabLockReadOnly: FC<Props> = ({ resource, id, children }) => {
  const t = useT();
  const { readOnly, holderUserId } = useForeignCollabLock(resource, id);

  return (
    <div
      style={{
        position: "relative",
        minHeight: 0,
        flex: 1,
        display: "flex",
        flexDirection: "column"
      }}
    >
      {readOnly && holderUserId ? (
        <div
          data-testid="collab-lock-banner"
          style={{
            flexShrink: 0,
            padding: "8px 10px",
            fontSize: 12,
            lineHeight: 1.35,
            background: "rgba(255, 170, 60, 0.18)",
            borderBottom: "1px solid rgba(255,170,60,0.35)"
          }}
        >
          {t("Collab object locked by another user", { userId: holderUserId })}
        </div>
      ) : null}
      <div
        style={{
          flex: 1,
          minHeight: 0,
          pointerEvents: readOnly ? "none" : undefined,
          opacity: readOnly ? 0.55 : undefined,
          overflow: "hidden"
        }}
      >
        {children}
      </div>
    </div>
  );
};

export default CollabLockReadOnly;
