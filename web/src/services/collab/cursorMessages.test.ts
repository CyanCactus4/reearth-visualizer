import { describe, expect, it } from "vitest";

import { cursorPayload } from "./cursorMessages";

describe("cursorPayload", () => {
  it("serializes normalized cursor", () => {
    const s = cursorPayload(0.5, 0.25, true);
    expect(JSON.parse(s)).toEqual({
      v: 1,
      t: "cursor",
      d: { x: 0.5, y: 0.25, inside: true }
    });
  });
});
