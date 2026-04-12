export type CollabActivityKind = "typing" | "move";

export function activityPayload(
  kind: CollabActivityKind,
  clientId?: string
): string {
  const d: Record<string, unknown> = { kind };
  if (clientId && clientId.length > 0) {
    d.clientId = clientId;
  }
  return JSON.stringify({ v: 1, t: "activity", d });
}
