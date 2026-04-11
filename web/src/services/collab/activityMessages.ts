export type CollabActivityKind = "typing" | "move";

export function activityPayload(kind: CollabActivityKind): string {
  return JSON.stringify({ v: 1, t: "activity", d: { kind } });
}
