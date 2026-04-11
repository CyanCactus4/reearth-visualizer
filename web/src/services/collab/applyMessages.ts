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

export function applyCreateStoryBlockPayload(params: {
  sceneId: string;
  storyId: string;
  pageId: string;
  pluginId: string;
  extensionId: string;
  index?: number;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "create_story_block",
    sceneId: params.sceneId,
    storyId: params.storyId,
    pageId: params.pageId,
    pluginId: params.pluginId,
    extensionId: params.extensionId
  };
  if (params.index != null) d.index = params.index;
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyRemoveStoryBlockPayload(params: {
  sceneId: string;
  storyId: string;
  pageId: string;
  blockId: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "remove_story_block",
    sceneId: params.sceneId,
    storyId: params.storyId,
    pageId: params.pageId,
    blockId: params.blockId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyCreateStoryPagePayload(params: {
  sceneId: string;
  storyId: string;
  title?: string;
  swipeable?: boolean;
  layers?: string[];
  swipeableLayers?: string[];
  index?: number;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "create_story_page",
    sceneId: params.sceneId,
    storyId: params.storyId
  };
  if (params.title != null) d.title = params.title;
  if (params.swipeable != null) d.swipeable = params.swipeable;
  if (params.layers != null && params.layers.length > 0) d.layers = params.layers;
  if (params.swipeableLayers != null && params.swipeableLayers.length > 0) {
    d.swipeableLayers = params.swipeableLayers;
  }
  if (params.index != null) d.index = params.index;
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyRemoveStoryPagePayload(params: {
  sceneId: string;
  storyId: string;
  pageId: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "remove_story_page",
    sceneId: params.sceneId,
    storyId: params.storyId,
    pageId: params.pageId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyMoveStoryPagePayload(params: {
  sceneId: string;
  storyId: string;
  pageId: string;
  index: number;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "move_story_page",
    sceneId: params.sceneId,
    storyId: params.storyId,
    pageId: params.pageId,
    index: params.index
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyDuplicateStoryPagePayload(params: {
  sceneId: string;
  storyId: string;
  pageId: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "duplicate_story_page",
    sceneId: params.sceneId,
    storyId: params.storyId,
    pageId: params.pageId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyUpdateStoryPagePayload(params: {
  sceneId: string;
  storyId: string;
  pageId: string;
  title?: string;
  swipeable?: boolean;
  /** When set (including `[]`), replaces page layers. Omit to leave unchanged. */
  layers?: string[];
  swipeableLayers?: string[];
  index?: number;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "update_story_page",
    sceneId: params.sceneId,
    storyId: params.storyId,
    pageId: params.pageId
  };
  if (params.title !== undefined) d.title = params.title;
  if (params.swipeable !== undefined) d.swipeable = params.swipeable;
  if (params.layers !== undefined) d.layers = params.layers;
  if (params.swipeableLayers !== undefined) {
    d.swipeableLayers = params.swipeableLayers;
  }
  if (params.index !== undefined) d.index = params.index;
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyAddNlsLayerSimplePayload(params: {
  sceneId: string;
  title: string;
  layerType: string;
  config?: unknown;
  index?: number;
  visible?: boolean;
  schema?: unknown;
  dataSourceName?: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "add_nls_layer_simple",
    sceneId: params.sceneId,
    title: params.title,
    layerType: params.layerType
  };
  if (params.config !== undefined) d.config = params.config;
  if (params.index !== undefined) d.index = params.index;
  if (params.visible !== undefined) d.visible = params.visible;
  if (params.schema !== undefined) d.schema = params.schema;
  if (params.dataSourceName !== undefined) d.dataSourceName = params.dataSourceName;
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyRemoveNlsLayerPayload(params: {
  sceneId: string;
  layerId: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "remove_nls_layer",
    sceneId: params.sceneId,
    layerId: params.layerId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyUpdateNlsLayerPayload(params: {
  sceneId: string;
  layerId: string;
  index?: number;
  name?: string;
  visible?: boolean;
  config?: unknown;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "update_nls_layer",
    sceneId: params.sceneId,
    layerId: params.layerId
  };
  if (params.index !== undefined) d.index = params.index;
  if (params.name !== undefined) d.name = params.name;
  if (params.visible !== undefined) d.visible = params.visible;
  if (params.config !== undefined) d.config = params.config;
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyUpdateNlsLayersPayload(params: {
  sceneId: string;
  layers: Array<{
    layerId: string;
    index?: number;
    name?: string;
    visible?: boolean;
    config?: unknown;
  }>;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "update_nls_layers",
    sceneId: params.sceneId,
    layers: params.layers
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyAddStylePayload(params: {
  sceneId: string;
  name: string;
  value: unknown;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "add_style",
    sceneId: params.sceneId,
    name: params.name,
    value: params.value
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyUpdateStylePayload(params: {
  sceneId: string;
  styleId: string;
  name?: string;
  value?: unknown;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "update_style",
    sceneId: params.sceneId,
    styleId: params.styleId
  };
  if (params.name !== undefined) d.name = params.name;
  if (params.value !== undefined) d.value = params.value;
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

export function applyRemoveStylePayload(params: {
  sceneId: string;
  styleId: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "remove_style",
    sceneId: params.sceneId,
    styleId: params.styleId
  };
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}

/** Collab apply for scene/widget/… property field edits (server `Property.UpdateValue`). */
export function applyUpdatePropertyValuePayload(params: {
  sceneId: string;
  propertyId: string;
  fieldId: string;
  /** GraphQL `ValueType` enum string, e.g. `STRING`, `NUMBER`. */
  type: string;
  value?: unknown;
  schemaGroupId?: string;
  itemId?: string;
  baseSceneRev?: number;
}): string {
  const d: Record<string, unknown> = {
    kind: "update_property_value",
    sceneId: params.sceneId,
    propertyId: params.propertyId,
    fieldId: params.fieldId,
    type: params.type
  };
  if (params.schemaGroupId !== undefined && params.schemaGroupId !== "") {
    d.schemaGroupId = params.schemaGroupId;
  }
  if (params.itemId !== undefined && params.itemId !== "") {
    d.itemId = params.itemId;
  }
  if (params.value !== undefined) d.value = params.value;
  if (params.baseSceneRev != null) d.baseSceneRev = params.baseSceneRev;
  return JSON.stringify({ v: 1, t: "apply", d });
}
