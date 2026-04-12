import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { CollabContextValue } from "@reearth/services/collab";

import CollabPresenceBar from "./CollabPresenceBar";

const sendRawM = vi.fn(() => true);
const sendChatM = vi.fn();

const collabState: CollabContextValue = {
  projectId: "proj-1",
  localUserId: undefined,
  localUserPhotoURL: undefined,
  status: "open",
  remoteUserPhotoURLs: {},
  presencePeerKeys: [],
  lastMessage: null,
  sendRaw: sendRawM,
  remoteCursors: {},
  remoteTypingUserIds: [],
  remoteMovingUserIds: [],
  resourceLocks: {},
  chatMessages: [],
  sendChat: sendChatM,
  remoteSceneRev: undefined,
  widgetEntityClocks: {},
  propertyFieldClocks: {},
  propertyDocClocks: {},
  collabReplicaId: "replica-test",
  tickPropertyFieldHlc: vi.fn(() => ({
    wall: 0,
    logical: 0,
    node: "n"
  }))
};

vi.mock("@reearth/services/collab", async (importOriginal) => {
  const mod = await importOriginal<typeof import("@reearth/services/collab")>();
  return {
    ...mod,
    useCollab: () => collabState
  };
});

describe("CollabPresenceBar", () => {
  beforeEach(() => {
    collabState.projectId = "proj-1";
    collabState.localUserId = undefined;
    collabState.status = "open";
    collabState.lastMessage = null;
    collabState.presencePeerKeys = [];
    collabState.remoteCursors = {};
    collabState.remoteTypingUserIds = [];
    collabState.remoteMovingUserIds = [];
    collabState.resourceLocks = {};
    collabState.chatMessages = [];
    sendRawM.mockClear();
    sendChatM.mockClear();
  });

  it("renders status when project is set", () => {
    collabState.lastMessage = null;
    render(<CollabPresenceBar />);
    expect(screen.getByTestId("collab-presence-bar")).toHaveTextContent(
      "Live: open"
    );
  });

  it("lists user from presencePeerKeys", () => {
    collabState.presencePeerKeys = ["alice"];
    render(<CollabPresenceBar />);
    expect(screen.getByTestId("collab-presence-bar")).toHaveTextContent("alice");
  });

  it("opens participants panel when toggle is clicked", async () => {
    const user = userEvent.setup();
    collabState.presencePeerKeys = ["alice"];
    render(<CollabPresenceBar />);
    await user.click(screen.getByTestId("collab-participants-toggle"));
    expect(screen.getByRole("dialog")).toBeTruthy();
  });

  it("shows typing peers from context", () => {
    collabState.remoteTypingUserIds = ["bob"];
    render(<CollabPresenceBar />);
    expect(screen.getByTestId("collab-presence-bar")).toHaveTextContent(
      "typing: bob"
    );
  });

  it("shows moving-map peers from context", () => {
    collabState.remoteMovingUserIds = ["carol"];
    render(<CollabPresenceBar />);
    expect(screen.getByTestId("collab-presence-bar")).toHaveTextContent(
      "moving map: carol"
    );
  });
});
