import { describe, expect, it } from "vitest";

import {
  isSameCollabTab,
  suppressCollabPeerAppliedNotification
} from "./peerInstanceKey";

describe("suppressCollabPeerAppliedNotification", () => {
  it("suppresses only when composite tab keys match", () => {
    expect(
      suppressCollabPeerAppliedNotification("u1", "tab-a", "u1", "tab-a")
    ).toBe(true);
    expect(
      suppressCollabPeerAppliedNotification("u1", "tab-b", "u1", "tab-a")
    ).toBe(false);
  });

  it("does not suppress when peer clientId is missing but local tab has an id", () => {
    expect(suppressCollabPeerAppliedNotification("u1", "tab-b", "u1")).toBe(
      false
    );
    expect(suppressCollabPeerAppliedNotification("u1", "tab-b", "u1", "")).toBe(
      false
    );
  });
});

describe("isSameCollabTab (typing/cursors)", () => {
  it("treats missing peer client as same user tab set", () => {
    expect(isSameCollabTab("u1", "t1", "u1", undefined)).toBe(true);
  });
});
