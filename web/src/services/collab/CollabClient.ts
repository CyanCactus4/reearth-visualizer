import { buildCollabWsUrl } from "./collabUrl";

export type CollabInbound =
  | { v: 1; t: "pong" }
  | {
      v: 1;
      t: "presence";
      d?: { event?: string; userId?: string; clientId?: string; photoURL?: string };
    }
  | {
      v: 1;
      t: "presence_snapshot";
      d?: {
        peers?: Array<{
          userId?: string;
          clientId?: string;
          photoURL?: string;
        }>;
      };
    }
  | {
      v: 1;
      t: "lock_changed";
      d?: {
        resource?: string;
        id?: string;
        holderUserId?: string;
        holderClientId?: string;
        until?: string;
        released?: boolean;
      };
    }
  | {
      v: 1;
      t: "lock_denied";
      d?: {
        resource?: string;
        id?: string;
        holderUserId?: string;
        holderClientId?: string;
        until?: string;
      };
    }
  | {
      v: 1;
      t: "chat";
      d?: {
        id?: string;
        userId?: string;
        text?: string;
        ts?: number;
        mentions?: string[];
      };
    }
  | {
      v: 1;
      t: "cursor";
      d?: {
        userId?: string;
        clientId?: string;
        x?: number;
        y?: number;
        inside?: boolean;
        ts?: number;
      };
    }
  | {
      v: 1;
      t: "activity";
      d?: {
        userId?: string;
        clientId?: string;
        kind?: string;
        ts?: number;
      };
    }
  | {
      v: 1;
      t: "applied";
      d?: {
        kind?: string;
        sceneId?: string;
        widgetId?: string;
        userId?: string;
        clientId?: string;
        sceneRev?: number;
        pluginId?: string;
        extensionId?: string;
        opKind?: string;
        layerId?: string;
        layerIds?: string[];
        blockId?: string;
        styleId?: string;
        propertyId?: string;
        fieldId?: string;
        itemId?: string;
        storyId?: string;
        pageId?: string;
      };
    }
  | {
      v: 1;
      t: "notify";
      d?: {
        kind?: string;
        fromUserId?: string;
        messageId?: string;
        text?: string;
        mentions?: string[];
      };
    }
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

  async connect(projectId: string, clientId?: string): Promise<WebSocket> {
    const token = await this.getAccessToken();
    const url = buildCollabWsUrl(
      this.apiBase,
      projectId,
      token ? token : undefined,
      clientId
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

  /** Returns false if the socket is not open (caller may queue offline). */
  sendRaw(json: string): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return false;
    }
    this.ws.send(json);
    return true;
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
