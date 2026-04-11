import type { WidgetAreaState } from "@reearth/services/state";

export type LockResource =
  | "layer"
  | "widget"
  | "scene"
  | "widget_area"
  | "style";

/** Stable lock id for a widget container area (must match server `widget_area` validation). */
export function widgetAreaLockId(area: WidgetAreaState): string {
  return `${area.zone}:${area.section}:${area.area}`;
}

export type LockAction = "acquire" | "release" | "heartbeat";

export function collabResourceLockKey(resource: LockResource, id: string): string {
  return `${resource}:${id}`;
}

export function lockPayload(
  action: LockAction,
  resource: LockResource,
  id: string
): string {
  return JSON.stringify({
    v: 1,
    t: "lock",
    d: { action, resource, id }
  });
}
