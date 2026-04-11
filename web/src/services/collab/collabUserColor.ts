/** Stable hue (0–360) from user id for cursor / presence accents. */
export function collabUserHue(userId: string): number {
  let h = 0;
  for (let i = 0; i < userId.length; i++) {
    h = (h * 31 + userId.charCodeAt(i)) >>> 0;
  }
  return h % 360;
}

export function collabUserColor(userId: string): string {
  const hue = collabUserHue(userId);
  return `hsl(${hue} 70% 55%)`;
}
