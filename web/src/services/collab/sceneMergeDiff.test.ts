import { describe, expect, it } from "vitest";

import { sceneMergeRichDiff } from "./sceneMergeDiff";

describe("sceneMergeRichDiff", () => {
  it("returns null when nodes are not scenes", () => {
    expect(sceneMergeRichDiff({}, {})).toBeNull();
  });

  it("detects widget add/remove/field change", () => {
    const cache = {
      node: {
        __typename: "Scene",
        widgets: [{ id: "w1", enabled: true, extended: false }],
        stories: []
      }
    };
    const network = {
      node: {
        __typename: "Scene",
        widgets: [
          { id: "w1", enabled: false, extended: false },
          { id: "w2", enabled: true, extended: true }
        ],
        stories: []
      }
    };
    const d = sceneMergeRichDiff(cache, network);
    expect(d).not.toBeNull();
    expect(d!.widgetDiff.added).toContain("w2");
    expect(d!.widgetDiff.changed.some((c) => c.id === "w1")).toBe(true);
  });
});
