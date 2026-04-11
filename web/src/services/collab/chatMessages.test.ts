import { describe, expect, it } from "vitest";

import { chatPayload } from "./chatMessages";

describe("chatPayload", () => {
  it("wraps text in collab chat envelope", () => {
    const s = chatPayload("hello");
    expect(JSON.parse(s)).toEqual({
      v: 1,
      t: "chat",
      d: { text: "hello" }
    });
  });
});
