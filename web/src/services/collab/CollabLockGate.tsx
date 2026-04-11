import { FC, ReactNode } from "react";

import CollabLockLeaseOnly from "./CollabLockLeaseOnly";
import CollabLockReadOnly from "./CollabLockReadOnly";
import type { LockResource } from "./lockMessages";

type Props = {
  resource: LockResource;
  id: string | undefined;
  children: ReactNode;
  /** When false, only read-only UI; lease must be held elsewhere (e.g. map-level layer lock). */
  manageLease?: boolean;
};

const CollabLockGate: FC<Props> = ({
  resource,
  id,
  children,
  manageLease = true
}) => {
  return (
    <>
      {manageLease ? (
        <CollabLockLeaseOnly resource={resource} id={id} active={!!id} />
      ) : null}
      <CollabLockReadOnly resource={resource} id={id}>
        {children}
      </CollabLockReadOnly>
    </>
  );
};

export default CollabLockGate;
