import { describe, expect, it } from "vitest";

import { parseApplyAuditResponse } from "./applyAuditApi";

describe("parseApplyAuditResponse", () => {
  it("returns empty for invalid bodies", () => {
    expect(parseApplyAuditResponse(null)).toEqual([]);
    expect(parseApplyAuditResponse({})).toEqual([]);
    expect(parseApplyAuditResponse({ entries: "x" })).toEqual([]);
  });

  it("parses collab_undo with opKind", () => {
    const rows = parseApplyAuditResponse({
      entries: [
        {
          id: "01",
          userId: "u1",
          userName: "Bob",
          kind: "collab_undo",
          opKind: "update_widget",
          sceneRev: 9,
          ts: 1
        }
      ]
    });
    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({
      kind: "collab_undo",
      opKind: "update_widget"
    });
  });

  it("parses rows with userName, layerId, and layerIds", () => {
    const rows = parseApplyAuditResponse({
      v: 1,
      entries: [
        {
          id: "64a1b2c3d4e5f67890123456",
          userId: "usr01",
          userName: "Ada",
          kind: "update_nls_layer",
          sceneRev: 42,
          ts: 1700000000000,
          sceneId: "sc01",
          layerId: "lay01"
        },
        {
          id: "64a1b2c3d4e5f67890123457",
          userId: "usr02",
          kind: "update_nls_layers",
          sceneRev: 43,
          ts: 1700000000001,
          layerIds: ["a", "b"]
        }
      ]
    });
    expect(rows).toHaveLength(2);
    expect(rows[0]).toMatchObject({
      id: "64a1b2c3d4e5f67890123456",
      userId: "usr01",
      userName: "Ada",
      kind: "update_nls_layer",
      sceneRev: 42,
      layerId: "lay01"
    });
    expect(rows[1].userName).toBeUndefined();
    expect(rows[1].layerIds).toEqual(["a", "b"]);
  });

  it("parses blockId for infobox / story apply rows", () => {
    const rows = parseApplyAuditResponse({
      entries: [
        {
          id: "01",
          userId: "u1",
          kind: "add_nls_infobox_block",
          sceneRev: 5,
          ts: 2,
          layerId: "lay01",
          blockId: "iblk01"
        }
      ]
    });
    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({
      kind: "add_nls_infobox_block",
      layerId: "lay01",
      blockId: "iblk01"
    });
  });

  it("drops rows missing required fields", () => {
    expect(
      parseApplyAuditResponse({
        entries: [{ id: "x", userId: "", kind: "k" }, { id: "", userId: "u", kind: "k" }]
      })
    ).toEqual([]);
  });
});
