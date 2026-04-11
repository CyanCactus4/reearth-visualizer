import { WidgetAlignSystemType } from "@reearth/services/gql";

/** Maps GraphQL enum to collab `apply` JSON (server parseAlignSystem). */
export function alignSystemForCollab(
  t: WidgetAlignSystemType
): "desktop" | "mobile" {
  return t === WidgetAlignSystemType.Mobile ? "mobile" : "desktop";
}

export function applyUpdateWidgetPayload(params: {
  sceneId: string;
  widgetId: string;
  type: WidgetAlignSystemType;
  /** Optional OT guard: must match server `scene.updatedAt` ms when sent. */
  baseSceneRev?: number;
  /** Optional per-field LWW clocks (`enabled`, `extended`, `layout`) from last `applied.entityClocks`. */
  entityClocks?: Record<string, number>;
  enabled?: boolean;
  location?: { zone: string; section: string; area: string };
  extended?: boolean;
  index?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "update_widget",
    sceneId: params.sceneId,
    alignSystem: alignSystemForCollab(params.type),
    widgetId: params.widgetId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  if (params.entityClocks && Object.keys(params.entityClocks).length > 0) {
    d.entityClocks = params.entityClocks;
  }
  if (params.enabled !== undefined) d.enabled = params.enabled;
  if (params.extended !== undefined) d.extended = params.extended;
  if (params.index !== undefined) d.index = params.index;
  if (params.location) {
    d.location = {
      zone: params.location.zone.toLowerCase(),
      section: params.location.section.toLowerCase(),
      area: params.location.area.toLowerCase()
    };
  }
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyRemoveWidgetPayload(params: {
  sceneId: string;
  widgetId: string;
  type: WidgetAlignSystemType;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "remove_widget",
    sceneId: params.sceneId,
    alignSystem: alignSystemForCollab(params.type),
    widgetId: params.widgetId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyAddWidgetPayload(params: {
  sceneId: string;
  type: WidgetAlignSystemType;
  pluginId: string;
  extensionId: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "add_widget",
    sceneId: params.sceneId,
    alignSystem: alignSystemForCollab(params.type),
    pluginId: params.pluginId,
    extensionId: params.extensionId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyMoveStoryBlockPayload(params: {
  sceneId: string;
  storyId: string;
  pageId: string;
  blockId: string;
  index: number;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "move_story_block",
    sceneId: params.sceneId,
    storyId: params.storyId,
    pageId: params.pageId,
    blockId: params.blockId,
    index: params.index
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}
