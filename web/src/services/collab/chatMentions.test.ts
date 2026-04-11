import { describe, expect, it } from "vitest";

import { extractChatMentions } from "./chatMentions";

describe("extractChatMentions", () => {
  it("returns empty when no mentions", () => {
    expect(extractChatMentions("hello", 10)).toEqual([]);
  });

  it("dedupes and caps", () => {
    expect(extractChatMentions("@a @b @a", 10)).toEqual(["a", "b"]);
    expect(extractChatMentions("@a @b @c", 2)).toEqual(["a", "b"]);
  });
});
