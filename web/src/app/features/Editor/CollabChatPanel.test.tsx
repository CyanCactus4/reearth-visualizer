import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { CollabStatus } from "@reearth/services/collab";

import CollabChatPanel from "./CollabChatPanel";

const collabState: {
  projectId: string | undefined;
  localUserId: string | undefined;
  status: CollabStatus;
  lastMessage: null;
  sendRaw: ReturnType<typeof vi.fn>;
  remoteCursors: Record<string, unknown>;
  remoteTypingUserIds: string[];
  resourceLocks: Record<string, unknown>;
  chatMessages: {
    id: string;
    userId: string;
    text: string;
    ts: number;
    pending?: boolean;
  }[];
  sendChat: ReturnType<typeof vi.fn>;
  remoteSceneRev: number | undefined;
} = {
  projectId: "proj-1",
  localUserId: "me",
  status: "open",
  lastMessage: null,
  sendRaw: vi.fn(() => true),
  remoteCursors: {} as Record<string, unknown>,
  remoteTypingUserIds: [] as string[],
  resourceLocks: {} as Record<string, unknown>,
  chatMessages: [] as {
    id: string;
    userId: string;
    text: string;
    ts: number;
    pending?: boolean;
  }[],
  sendChat: vi.fn(),
  remoteSceneRev: undefined as number | undefined
};

vi.mock("@reearth/services/collab", () => ({
  useCollab: () => collabState
}));

vi.mock("@reearth/services/i18n/hooks", () => ({
  useT: () => (key: string) => key
}));

describe("CollabChatPanel", () => {
  beforeEach(() => {
    collabState.projectId = "proj-1";
    collabState.status = "open";
    collabState.chatMessages = [];
    collabState.sendChat.mockClear();
  });

  it("renders nothing without projectId", () => {
    collabState.projectId = undefined;
    const { container } = render(<CollabChatPanel />);
    expect(container.firstChild).toBeNull();
  });

  it("renders chat lines and pending hint", () => {
    collabState.chatMessages = [
      { id: "a", userId: "u1", text: "hello", ts: 1 },
      { id: "b", userId: "me", text: "wait", ts: 2, pending: true }
    ];
    render(<CollabChatPanel />);
    expect(screen.getByTestId("collab-chat-panel")).toBeInTheDocument();
    expect(screen.getByTestId("collab-chat-line-a")).toHaveTextContent(
      "u1: hello"
    );
    expect(screen.getByTestId("collab-chat-line-b")).toHaveTextContent("…");
  });

  it("sends trimmed text on button click", async () => {
    const user = userEvent.setup();
    render(<CollabChatPanel />);
    await user.type(
      screen.getByPlaceholderText("Collab chat placeholder"),
      "  hi there  "
    );
    await user.click(screen.getByRole("button", { name: "Collab chat send" }));
    expect(collabState.sendChat).toHaveBeenCalledWith("hi there");
  });

  it("disables send when socket is not open", () => {
    collabState.status = "connecting";
    render(<CollabChatPanel />);
    expect(screen.getByRole("button", { name: "Collab chat send" })).toBeDisabled();
  });
});
