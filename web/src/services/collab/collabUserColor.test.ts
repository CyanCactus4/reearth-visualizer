import { describe, expect, it } from "vitest";

import {
  collabUserAvatarLetter,
  collabUserColor,
  collabUserHue
} from "./collabUserColor";

describe("collabUserColor", () => {
  it("returns stable hsl for id", () => {
    expect(collabUserHue("a")).toBe(collabUserHue("a"));
    expect(collabUserColor("user-1")).toMatch(/^hsl\(/);
  });
});

describe("collabUserAvatarLetter", () => {
  it("uses two letters from split id", () => {
    expect(collabUserAvatarLetter("alice.bob")).toBe("AB");
  });

  it("uses local part of email", () => {
    expect(collabUserAvatarLetter("carol.dev@example.com")).toBe("CD");
  });

  it("falls back to first alphanumerics of single token", () => {
    expect(collabUserAvatarLetter("peer")).toBe("PE");
  });
});
