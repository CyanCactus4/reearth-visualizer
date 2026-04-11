const mentionRe = /@([a-zA-Z0-9_-]+)/g;

/** Same rules as server `ExtractChatMentions` (dedupe, order, cap). */
export function extractChatMentions(text: string, max: number): string[] {
  if (!text || max <= 0) return [];
  const seen = new Set<string>();
  const out: string[] = [];
  mentionRe.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = mentionRe.exec(text)) !== null) {
    const name = m[1];
    if (!name || seen.has(name)) continue;
    seen.add(name);
    out.push(name);
    if (out.length >= max) break;
  }
  return out;
}
