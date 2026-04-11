import {
  useCollabLockLease,
  useForeignCollabLock,
  type LockResource
} from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { FC, ReactNode } from "react";

type Props = {
  resource: LockResource;
  id: string | undefined;
  children: ReactNode;
};

/**
 * Holds a collab edit lease for the selected object and dims the inspector when locked by a peer.
 */
const CollabLockGate: FC<Props> = ({ resource, id, children }) => {
  const t = useT();
  const active = !!id;
  useCollabLockLease(resource, id, active);
  const { readOnly, holderUserId } = useForeignCollabLock(resource, id);

  return (
    <div style={{ position: "relative", minHeight: 0, flex: 1, display: "flex", flexDirection: "column" }}>
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

export default CollabLockGate;
