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
});
