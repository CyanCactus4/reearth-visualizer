/** Outbound cursor position in normalized viewport coordinates [0,1]. */
export function cursorPayload(
  x: number,
  y: number,
  inside: boolean,
  clientId?: string
): string {
  const d: Record<string, unknown> = { x, y, inside };
  if (clientId && clientId.length > 0) {
    d.clientId = clientId;
  }
  return JSON.stringify({
    v: 1,
    t: "cursor",
    d
  });
}
