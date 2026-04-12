import { describe, expect, it } from "vitest";

import { COLLAB_APPLY_FAILURE_SCENE_REFETCH_CODES } from "./collabApplyErrorCodes";

describe("COLLAB_APPLY_FAILURE_SCENE_REFETCH_CODES", () => {
  it("includes stale_property_field for per-field LWW refetch", () => {
    expect(COLLAB_APPLY_FAILURE_SCENE_REFETCH_CODES.has("stale_property_field")).toBe(
      true
    );
  });

  it("includes stale_property_doc for merge_property_json CAS", () => {
    expect(COLLAB_APPLY_FAILURE_SCENE_REFETCH_CODES.has("stale_property_doc")).toBe(
      true
    );
  });
});
