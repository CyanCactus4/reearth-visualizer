import { describe, expect, it } from "vitest";

import { activityPayload } from "./activityMessages";

describe("activityPayload", () => {
  it("wraps kind", () => {
    expect(JSON.parse(activityPayload("typing"))).toEqual({
      v: 1,
      t: "activity",
      d: { kind: "typing" }
    });
  });

  it("includes clientId when provided", () => {
    expect(JSON.parse(activityPayload("move", "cid"))).toEqual({
      v: 1,
      t: "activity",
      d: { kind: "move", clientId: "cid" }
    });
  });
});
