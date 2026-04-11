import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CollabPresenceBar from "./CollabPresenceBar";

const collabState = {
  projectId: "proj-1",
  localUserId: undefined as string | undefined,
  status: "open" as const,
  lastMessage: null as { v: 1; t: string; d?: Record<string, string> } | null,
  sendRaw: vi.fn(),
  remoteCursors: {} as Record<string, { x: number; y: number; inside: boolean; ts: number }>,
  remoteTypingUserIds: [] as string[],
  resourceLocks: {} as Record<string, { holderUserId: string; until?: string }>
};

vi.mock("@reearth/services/collab", () => ({
  useCollab: () => collabState
}));

describe("CollabPresenceBar", () => {
  beforeEach(() => {
    collabState.projectId = "proj-1";
    collabState.localUserId = undefined;
    collabState.status = "open";
    collabState.lastMessage = null;
    collabState.remoteCursors = {};
    collabState.remoteTypingUserIds = [];
    collabState.resourceLocks = {};
    collabState.sendRaw.mockClear();
  });

  it("renders status when project is set", () => {
    collabState.lastMessage = null;
    render(<CollabPresenceBar />);
    expect(screen.getByTestId("collab-presence-bar")).toHaveTextContent(
      "Live: open"
    );
  });

  it("lists user after presence join", () => {
    collabState.lastMessage = {
      v: 1,
      t: "presence",
      d: { event: "join", userId: "alice" }
    };
    render(<CollabPresenceBar />);
    expect(screen.getByTestId("collab-presence-bar")).toHaveTextContent("alice");
  });

  it("shows typing peers from context", () => {
    collabState.remoteTypingUserIds = ["bob"];
    render(<CollabPresenceBar />);
    expect(screen.getByTestId("collab-presence-bar")).toHaveTextContent(
      "typing: bob"
    );
  });
});
