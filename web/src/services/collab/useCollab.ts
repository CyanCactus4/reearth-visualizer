import { useContext } from "react";

import { CollabContext, type CollabContextValue } from "./collabContext";

export const useCollab = (): CollabContextValue | null =>
  useContext(CollabContext);
