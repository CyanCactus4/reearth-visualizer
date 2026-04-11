export type LockResource = "layer" | "widget";

export type LockAction = "acquire" | "release" | "heartbeat";

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
