import { describe, expect, it } from "vitest";

import { lockPayload } from "./lockMessages";

describe("lockPayload", () => {
  it("serializes acquire for layer", () => {
    const s = lockPayload("acquire", "layer", "abc");
    expect(JSON.parse(s)).toEqual({
      v: 1,
      t: "lock",
      d: { action: "acquire", resource: "layer", id: "abc" }
    });
  });
});
