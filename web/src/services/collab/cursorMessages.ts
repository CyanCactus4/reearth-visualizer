/** Outbound cursor position in normalized viewport coordinates [0,1]. */
export function cursorPayload(x: number, y: number, inside: boolean): string {
  return JSON.stringify({
    v: 1,
    t: "cursor",
    d: { x, y, inside }
  });
}
