import { describe, expect, it, vi } from "vitest";

import {
  collabApplyKindLabel,
  collabPeerAppliedAction,
  collabPeerAppliedTargetHint,
  formatCollabAuditTimestamp
} from "./collabApplyPeerToast";

describe("collabApplyPeerToast", () => {
  const t = vi.fn((key: string) => key);

  it("collabPeerAppliedTargetHint aggregates ids", () => {
    expect(
      collabPeerAppliedTargetHint({
        layerId: "layer-long-id-here"
      })
    ).toContain("layer");
    expect(
      collabPeerAppliedTargetHint({
        widgetId: "w1",
        layerIds: ["a", "b"]
      })
    ).toContain("2 layers");
  });

  it("collabApplyKindLabel falls back to humanized kind when missing i18n", () => {
    expect(collabApplyKindLabel("foo_bar_kind", t)).toBe("foo bar kind");
  });

  it("formatCollabAuditTimestamp returns em dash for missing ts", () => {
    expect(formatCollabAuditTimestamp(0, "en")).toBe("—");
    expect(formatCollabAuditTimestamp(-1, "en")).toBe("—");
    const sec = 1_700_000_000;
    const s = formatCollabAuditTimestamp(sec, "en");
    expect(s).not.toBe("—");
    expect(s.length).toBeGreaterThan(4);
  });

  it("formatCollabAuditTimestamp accepts unix ms", () => {
    const ms = 1_700_000_000_000;
    expect(formatCollabAuditTimestamp(ms, "en")).toMatch(/\d/);
  });

  it("collabPeerAppliedAction uses undo meta", () => {
    const tm = vi.fn((key: string, opts?: { op?: string }) => {
      if (key === "Collab apply meta undo") return `UNDO ${opts?.op ?? ""}`;
      return key;
    });
    expect(
      collabPeerAppliedAction(
        "collab_undo",
        "update_widget",
        tm as (a: string, b?: { op?: string }) => string
      )
    ).toContain("UNDO");
  });
});
