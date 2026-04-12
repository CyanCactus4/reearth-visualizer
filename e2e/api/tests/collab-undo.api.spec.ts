import fs from "fs";
import path from "path";

import { API_BASE_URL } from "../config/env";
import { test, expect } from "../fixtures/api-test-fixtures";
import {
  ADD_NLS_LAYER_SIMPLE,
  CREATE_PROJECT,
  CREATE_SCENE,
  DELETE_PROJECT
} from "../graphql/mutations";
import { GET_ME } from "../graphql/queries";

const tokenPath = path.join(__dirname, "../../.auth/api-token.json");

test.describe.configure({ mode: "serial" });

test.describe("Collab REST: undo / admin restore contract", () => {
  let projectId: string;
  let sceneId: string;

  test.afterAll(async ({ gqlClient }) => {
    if (!projectId) return;
    try {
      await gqlClient.mutate(DELETE_PROJECT, { input: { projectId } });
    } catch {
      // ignore
    }
  });

  test("Setup: project and scene for collab sceneId", async ({ gqlClient }) => {
    const { data: me } = await gqlClient.query<{
      me: { myWorkspaceId: string };
    }>(GET_ME);

    const { data: proj } = await gqlClient.mutate<{
      createProject: { project: { id: string } };
    }>(CREATE_PROJECT, {
      input: {
        workspaceId: me.me.myWorkspaceId,
        visualizer: "CESIUM",
        name: "Collab API smoke",
        coreSupport: true
      }
    });
    projectId = proj.createProject.project.id;

    const { data: sc } = await gqlClient.mutate<{
      createScene: { scene: { id: string } };
    }>(CREATE_SCENE, { input: { projectId } });
    sceneId = sc.createScene.scene.id;

    await gqlClient.mutate(ADD_NLS_LAYER_SIMPLE, {
      input: {
        sceneId,
        layerType: "simple",
        title: "Layer",
        visible: true,
        config: { data: { type: "geojson" } }
      }
    });
  });

  test("POST /api/collab/undo without auth returns 401", async ({ request }) => {
    const res = await request.post(`${API_BASE_URL}/api/collab/undo`, {
      headers: { "Content-Type": "application/json" },
      data: { sceneId }
    });
    expect(res.status()).toBe(401);
  });

  test("POST /api/collab/undo with auth: empty stack (400) or undo disabled (404)", async ({
    request
  }) => {
    const { token } = JSON.parse(fs.readFileSync(tokenPath, "utf-8"));
    const res = await request.post(`${API_BASE_URL}/api/collab/undo`, {
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`
      },
      data: { sceneId }
    });
    const status = res.status();
    expect([400, 404]).toContain(status);
    if (status === 400) {
      const text = (await res.text()).toLowerCase();
      expect(text).toContain("nothing");
    }
  });

  test("POST /api/collab/admin/restore-scene: 501 placeholder or 404 if collab off", async ({
    request
  }) => {
    const { token } = JSON.parse(fs.readFileSync(tokenPath, "utf-8"));
    const res = await request.post(
      `${API_BASE_URL}/api/collab/admin/restore-scene`,
      {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`
        },
        data: { sceneId, targetSceneRev: 1 }
      }
    );
    expect([200, 400, 403, 404, 501]).toContain(res.status());
    if (res.status() === 501) {
      const body = await res.json();
      expect(body).toHaveProperty("error");
    }
  });
});
