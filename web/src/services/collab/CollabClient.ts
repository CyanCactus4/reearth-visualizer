import { buildCollabWsUrl } from "./collabUrl";

export type CollabInbound =
  | { v: 1; t: "pong" }
  | { v: 1; t: "presence"; d?: { event?: string; userId?: string } }
  | { v: 1; t: "lock_changed"; d?: Record<string, unknown> }
  | { v: 1; t: "lock_denied"; d?: Record<string, unknown> }
  | { v: 1; t: "applied"; d?: Record<string, unknown> }
  | { v: 1; t: "error"; d?: { code?: string; message?: string } }
  | { v: 1; t: string; d?: unknown };

/** Thin WebSocket helper for /api/collab/ws (ping + optional apply relay). */
export class CollabClient {
  private ws: WebSocket | null = null;

  constructor(
    private readonly apiBase: string,
    private readonly getAccessToken: () => Promise<string>
  ) {}

  get socket(): WebSocket | null {
    return this.ws;
  }

  async connect(projectId: string): Promise<WebSocket> {
    const token = await this.getAccessToken();
    const url = buildCollabWsUrl(
      this.apiBase,
      projectId,
      token ? token : undefined
    );
    this.ws = new WebSocket(url);
    return this.ws;
  }

  disconnect(): void {
    this.ws?.close();
    this.ws = null;
  }

  ping(): void {
    this.ws?.send(JSON.stringify({ v: 1, t: "ping" }));
  }

  sendRaw(json: string): void {
    this.ws?.send(json);
  }

  onMessage(handler: (msg: CollabInbound) => void): void {
    if (!this.ws) return;
    this.ws.onmessage = (ev: MessageEvent<string>) => {
      try {
        const data = JSON.parse(ev.data) as CollabInbound;
        handler(data);
      } catch {
        /* ignore malformed */
      }
    };
  }
}
