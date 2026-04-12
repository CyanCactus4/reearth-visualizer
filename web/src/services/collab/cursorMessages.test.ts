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

  it("includes clientId when provided", () => {
    const s = cursorPayload(0.1, 0.2, false, "tab-a");
    expect(JSON.parse(s)).toEqual({
      v: 1,
      t: "cursor",
      d: { x: 0.1, y: 0.2, inside: false, clientId: "tab-a" }
    });
  });
});
