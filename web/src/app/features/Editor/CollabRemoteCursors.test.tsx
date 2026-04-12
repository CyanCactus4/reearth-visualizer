import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import CollabRemoteCursors from "./CollabRemoteCursors";

const collabState = {
  projectId: "p1",
  localUserId: "me",
  status: "open" as const,
  lastMessage: null,
  sendRaw: vi.fn(),
  remoteCursors: {} as Record<
    string,
    { x: number; y: number; inside: boolean; ts: number }
  >,
  remoteTypingUserIds: [] as string[],
  resourceLocks: {} as Record<string, { holderUserId: string; until?: string }>
};

vi.mock("@reearth/services/collab", () => ({
  useCollab: () => collabState,
  collabUserColor: (id: string) => `color-${id}`,
  collabUserAvatarLetter: (id: string) => `AV-${id}`
}));

describe("CollabRemoteCursors", () => {
  it("renders nothing without cursors", () => {
    collabState.remoteCursors = {};
    const { container } = render(<CollabRemoteCursors />);
    expect(container.firstChild).toBeNull();
  });

  it("renders marker for inside peer", () => {
    collabState.remoteCursors = {
      peer: { x: 0.5, y: 0.5, inside: true, ts: Date.now() }
    };
    const { container } = render(<CollabRemoteCursors />);
    expect(screen.getByTestId("collab-remote-cursors")).toBeInTheDocument();
    expect(screen.getByText("peer")).toBeInTheDocument();
    expect(container.querySelector('[title="peer"]')).toBeTruthy();
    expect(screen.getByTestId("collab-cursor-avatar-peer")).toHaveTextContent(
      "AV-peer"
    );
    expect(screen.getByLabelText("Avatar of peer")).toBeInTheDocument();
  });
});
