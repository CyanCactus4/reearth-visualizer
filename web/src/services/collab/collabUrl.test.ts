import { describe, expect, it, vi } from "vitest";

import {
  buildCollabApplyAuditUrl,
  buildCollabChatUrl,
  buildCollabRedoPostUrl,
  buildCollabUndoPostUrl,
  buildCollabWsUrl,
  postCollabRedo,
  postCollabUndo
} from "./collabUrl";

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

  it("appends clientId when provided", () => {
    const u = buildCollabWsUrl(
      "http://localhost:8080/api",
      "proj1",
      undefined,
      "replica-uuid-1"
    );
    expect(u).toContain("projectId=proj1");
    expect(u).toContain("clientId=replica-uuid-1");
  });
});

describe("buildCollabChatUrl", () => {
  it("builds REST collab chat URL with projectId and limit", () => {
    expect(
      buildCollabChatUrl("http://localhost:8080/api", "proj1", 50)
    ).toBe("http://localhost:8080/api/collab/chat?projectId=proj1&limit=50");
  });
});

describe("buildCollabApplyAuditUrl", () => {
  it("builds REST apply-audit URL", () => {
    expect(
      buildCollabApplyAuditUrl("http://localhost:8080/api", "proj1", 20)
    ).toBe(
      "http://localhost:8080/api/collab/apply-audit?projectId=proj1&limit=20"
    );
  });

  it("appends sceneId when provided", () => {
    const u = buildCollabApplyAuditUrl(
      "http://localhost:8080/api",
      "proj1",
      10,
      "01fbpdqax0ttrftj3gb5gm4rw7"
    );
    expect(u).toContain("projectId=proj1");
    expect(u).toContain("limit=10");
    expect(u).toContain(
      "sceneId=01fbpdqax0ttrftj3gb5gm4rw7"
    );
  });
});

describe("buildCollabUndoPostUrl / buildCollabRedoPostUrl", () => {
  it("builds POST URLs under api base", () => {
    expect(buildCollabUndoPostUrl("http://localhost:8080/api")).toBe(
      "http://localhost:8080/api/collab/undo"
    );
    expect(buildCollabRedoPostUrl("https://x.test/api/")).toBe(
      "https://x.test/api/collab/redo"
    );
  });
});

describe("postCollabUndo / postCollabRedo", () => {
  it("POSTs JSON sceneId and optional bearer token", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(null, { status: 200 })
    );
    try {
      await postCollabUndo(
        "http://localhost:8080/api",
        async () => "tok",
        "scene-1"
      );
      expect(fetchSpy).toHaveBeenCalledTimes(1);
      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toBe("http://localhost:8080/api/collab/undo");
      expect(init?.method).toBe("POST");
      expect(init?.credentials).toBeUndefined();
      expect((init?.headers as Record<string, string>).Authorization).toBe(
        "Bearer tok"
      );
      expect(init?.body).toBe(JSON.stringify({ sceneId: "scene-1" }));

      await postCollabRedo(
        "http://localhost:8080/api",
        async () => null,
        "scene-2"
      );
      const init2 = fetchSpy.mock.calls[1][1] as RequestInit;
      expect(init2?.body).toBe(JSON.stringify({ sceneId: "scene-2" }));
      expect(
        (init2.headers as Record<string, string>).Authorization
      ).toBeUndefined();
    } finally {
      fetchSpy.mockRestore();
    }
  });
});
