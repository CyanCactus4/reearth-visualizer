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
}): string {
  return JSON.stringify({
    v: 1,
    t: "apply",
    d: {
      kind: "remove_widget",
      sceneId: params.sceneId,
      alignSystem: alignSystemForCollab(params.type),
      widgetId: params.widgetId
    }
  });
}

export function applyAddWidgetPayload(params: {
  sceneId: string;
  type: WidgetAlignSystemType;
  pluginId: string;
  extensionId: string;
}): string {
  return JSON.stringify({
    v: 1,
    t: "apply",
    d: {
      kind: "add_widget",
      sceneId: params.sceneId,
      alignSystem: alignSystemForCollab(params.type),
      pluginId: params.pluginId,
      extensionId: params.extensionId
    }
  });
}
