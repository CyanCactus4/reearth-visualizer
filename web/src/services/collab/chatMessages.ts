export function chatPayload(text: string): string {
  return JSON.stringify({ v: 1, t: "chat", d: { text } });
}
