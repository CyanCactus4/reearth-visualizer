import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CollabPresenceBar from "./CollabPresenceBar";

const collabState = {
  projectId: "proj-1",
  status: "open" as const,
  lastMessage: null as { v: 1; t: string; d?: Record<string, string> } | null,
  sendRaw: vi.fn()
};

vi.mock("@reearth/services/collab", () => ({
  useCollab: () => collabState
}));

describe("CollabPresenceBar", () => {
  beforeEach(() => {
    collabState.projectId = "proj-1";
    collabState.status = "open";
    collabState.lastMessage = null;
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
});
