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

function pickAlphaNumUpper(s: string, take: number): string {
  const out: string[] = [];
  for (let i = 0; i < s.length && out.length < take; i++) {
    const ch = s[i]!;
    if (/[a-zA-Z0-9]/.test(ch)) out.push(ch.toUpperCase());
  }
  return out.join("");
}

/** Up to two initials for a minimal on-map “avatar” chip (TASK.md FR-3: names / avatars). */
export function collabUserAvatarLetter(userId: string): string {
  const s = userId.trim();
  if (!s) return "?";
  const local = s.includes("@") ? s.slice(0, s.indexOf("@")) : s;
  const parts = local.split(/[\s._\-/+]+/).filter(Boolean);
  if (parts.length >= 2) {
    const a = pickAlphaNumUpper(parts[0]!, 1);
    const b = pickAlphaNumUpper(parts[1]!, 1);
    if (a && b) return a + b;
  }
  const mono = pickAlphaNumUpper(parts[0] ?? local, 2);
  return mono || "?";
}
