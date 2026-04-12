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
import { GET_ME, GET_SCENE } from "../graphql/queries";

const tokenPath = path.join(__dirname, "../../.auth/api-token.json");

test.describe.configure({ mode: "serial" });

type FieldPick = {
  propertyId: string;
  fieldId: string;
  schemaGroupId?: string;
  itemId?: string;
  type: string;
  value: unknown;
};

function wsCollabUrl(projectId: string, accessToken: string): string {
  const origin = API_BASE_URL.replace(/\/$/, "");
  const wsOrigin = origin
    .replace(/^https:\/\//, "wss://")
    .replace(/^http:\/\//, "ws://");
  const q = new URLSearchParams({
    projectId,
    access_token: accessToken
  });
  return `${wsOrigin}/api/collab/ws?${q.toString()}`;
}

function pickUpdatableSceneField(node: unknown): FieldPick | null {
  const n = node as {
    property?: {
      id?: string;
      items?: Array<{
        schemaGroupId?: string;
        id?: string;
        fields?: Array<{ fieldId?: string; type?: string; value?: unknown }>;
        groups?: Array<{
          id?: string;
          schemaGroupId?: string;
          fields?: Array<{ fieldId?: string; type?: string; value?: unknown }>;
        }>;
      }>;
    };
  };
  const prop = n?.property;
  if (!prop?.id || !Array.isArray(prop.items)) return null;
  for (const item of prop.items) {
    const sg = item.schemaGroupId;
    if (item.fields?.length) {
      for (const f of item.fields) {
        const pick = toPick(prop.id, f, sg, undefined);
        if (pick) return pick;
      }
    }
    if (item.groups?.length) {
      for (const g of item.groups) {
        if (!g.fields?.length) continue;
        for (const f of g.fields) {
          const pick = toPick(prop.id, f, sg, g.id);
          if (pick) return pick;
        }
      }
    }
  }
  return null;
}

function toPick(
  propertyId: string,
  f: { fieldId?: string; type?: string; value?: unknown },
  schemaGroupId?: string,
  itemId?: string
): FieldPick | null {
  if (!f.fieldId || !f.type) return null;
  const t = String(f.type).toUpperCase();
  if (t === "BOOL") {
    return {
      propertyId,
      fieldId: f.fieldId,
      schemaGroupId,
      itemId,
      type: "BOOL",
      value: !(f.value === true)
    };
  }
  if (t === "NUMBER") {
    const cur = typeof f.value === "number" ? f.value : 0;
    return {
      propertyId,
      fieldId: f.fieldId,
      schemaGroupId,
      itemId,
      type: "NUMBER",
      value: cur + 0.0001
    };
  }
  if (t === "STRING") {
    const cur = typeof f.value === "string" ? f.value : "";
    return {
      propertyId,
      fieldId: f.fieldId,
      schemaGroupId,
      itemId,
      type: "STRING",
      value: `${cur}`.length ? `${cur}·` : "e2e"
    };
  }
  return null;
}

function awaitApplied(
  ws: WebSocket,
  timeoutMs: number
): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    const t = setTimeout(() => {
      try {
        ws.close();
      } catch {
        /* ignore */
      }
      reject(new Error("timeout waiting for applied"));
    }, timeoutMs);
    ws.onmessage = (ev: { data: unknown }) => {
      try {
        const raw = typeof ev.data === "string" ? ev.data : String(ev.data);
        const j = JSON.parse(raw) as {
          t?: string;
          d?: Record<string, unknown>;
        };
        if (j.t === "applied" && j.d?.kind === "update_property_value") {
          clearTimeout(t);
          ws.onmessage = null;
          resolve(j.d ?? {});
        }
      } catch {
        /* ignore */
      }
    };
  });
}

test.describe("Collab WebSocket: two clients, apply → applied", () => {
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

  test("two sockets: one apply update_property_value, both see applied", async ({
    gqlClient
  }) => {
    const { token } = JSON.parse(fs.readFileSync(tokenPath, "utf-8"));

    const { data: me } = await gqlClient.query<{
      me: { myWorkspaceId: string };
    }>(GET_ME);

    const { data: proj } = await gqlClient.mutate<{
      createProject: { project: { id: string } };
    }>(CREATE_PROJECT, {
      input: {
        workspaceId: me.me.myWorkspaceId,
        visualizer: "CESIUM",
        name: "Collab WS two-client",
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

    const { data: sceneData } = await gqlClient.query<{
      node?: unknown;
    }>(GET_SCENE, { sceneId });
    const field = pickUpdatableSceneField(sceneData?.node);
    if (!field) {
      test.skip(true, "no BOOL/NUMBER/STRING field on scene property");
      return;
    }

    const url = wsCollabUrl(projectId, token);
    const ws1 = new WebSocket(url);
    const ws2 = new WebSocket(url);

    await Promise.all([
      new Promise<void>((res, rej) => {
        ws1.onopen = () => res();
        ws1.onerror = () => rej(new Error("ws1 connect failed"));
      }),
      new Promise<void>((res, rej) => {
        ws2.onopen = () => res();
        ws2.onerror = () => rej(new Error("ws2 connect failed"));
      })
    ]);

    const applied2 = awaitApplied(ws2, 25000);

    const now = Date.now();
    const d: Record<string, unknown> = {
      kind: "update_property_value",
      sceneId,
      propertyId: field.propertyId,
      fieldId: field.fieldId,
      type: field.type,
      value: field.value,
      fieldHlc: { wall: now, logical: 0, node: "e2e-ws-two-clients" }
    };
    if (field.schemaGroupId) d.schemaGroupId = field.schemaGroupId;
    if (field.itemId) d.itemId = field.itemId;

    ws1.send(JSON.stringify({ v: 1, t: "apply", d }));

    const d2 = await applied2;
    expect(d2.kind).toBe("update_property_value");
    expect(typeof d2.sceneRev).toBe("number");

    ws1.close();
    ws2.close();
  });
});
