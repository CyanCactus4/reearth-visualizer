import { describe, expect, it } from "vitest";

import { collabUserColor, collabUserHue } from "./collabUserColor";

describe("collabUserColor", () => {
  it("returns stable hsl for id", () => {
    expect(collabUserHue("a")).toBe(collabUserHue("a"));
    expect(collabUserColor("user-1")).toMatch(/^hsl\(/);
  });
});
