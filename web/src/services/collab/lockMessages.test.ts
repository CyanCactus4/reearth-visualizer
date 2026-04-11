import { describe, expect, it } from "vitest";

import {
  collabResourceLockKey,
  lockPayload,
  widgetAreaLockId
} from "./lockMessages";

describe("collabResourceLockKey", () => {
  it("builds stable keys", () => {
    expect(collabResourceLockKey("layer", "x")).toBe("layer:x");
    expect(collabResourceLockKey("widget", "w")).toBe("widget:w");
  });
});

describe("widgetAreaLockId", () => {
  it("formats zone:section:area", () => {
    expect(
      widgetAreaLockId({
        zone: "inner",
        section: "center",
        area: "middle"
      })
    ).toBe("inner:center:middle");
  });
});

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
