import { FC } from "react";

import type { LockResource } from "./lockMessages";
import { useCollabLockLease } from "./useCollabResourceLock";

type Props = {
  resource: LockResource;
  id: string | undefined;
  active: boolean;
};

/** Acquires / heartbeats / releases collab lock; renders nothing (map-level lease). */
const CollabLockLeaseOnly: FC<Props> = ({ resource, id, active }) => {
  useCollabLockLease(resource, id, active);
  return null;
};

export default CollabLockLeaseOnly;
