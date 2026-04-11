import { describe, expect, it } from "vitest";

import { buildCollabWsUrl } from "./collabUrl";

describe("buildCollabWsUrl", () => {
  it("maps http API to ws collab path", () => {
    expect(buildCollabWsUrl("http://localhost:8080/api", "proj1")).toBe(
      "ws://localhost:8080/api/collab/ws?projectId=proj1"
    );
  });

  it("maps https to wss and appends token", () => {
    const u = buildCollabWsUrl(
      "https://example.com/api/",
      "p-2",
      "tok%3D"
    );
    expect(u.startsWith("wss://example.com/api/collab/ws?")).toBe(true);
    expect(u).toContain("projectId=p-2");
    expect(u).toContain("access_token=");
  });
});
