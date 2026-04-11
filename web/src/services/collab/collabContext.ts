import { createContext } from "react";

import type { CollabInbound } from "./CollabClient";

export type CollabStatus = "idle" | "connecting" | "open" | "closed" | "error";

export type CollabContextValue = {
  status: CollabStatus;
  projectId: string | undefined;
  lastMessage: CollabInbound | null;
  sendRaw: (json: string) => void;
};

export const CollabContext = createContext<CollabContextValue | null>(null);
