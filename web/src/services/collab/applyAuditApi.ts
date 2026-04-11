/** One row from GET /api/collab/apply-audit (server collab.ApplyAuditListRow). */
export type CollabApplyAuditEntry = {
  id: string;
  userId: string;
  userName?: string;
  kind: string;
  /** Domain stack kind when kind is collab_undo / collab_redo (e.g. update_widget). */
  opKind?: string;
  sceneRev: number;
  ts: number;
  sceneId?: string;
  widgetId?: string;
  storyId?: string;
  pageId?: string;
  blockId?: string;
  propertyId?: string;
  fieldId?: string;
  styleId?: string;
  layerId?: string;
  layerIds?: string[];
};

function str(v: unknown): string | undefined {
  return typeof v === "string" ? v : undefined;
}

function num(v: unknown): number {
  if (typeof v === "number" && !Number.isNaN(v)) return v;
  if (typeof v === "string") {
    const n = Number(v);
    if (!Number.isNaN(n)) return n;
  }
  return 0;
}

function strList(v: unknown): string[] | undefined {
  if (!Array.isArray(v)) return undefined;
  const out = v.filter((x): x is string => typeof x === "string");
  return out.length > 0 ? out : undefined;
}

function parseEntry(raw: unknown): CollabApplyAuditEntry | null {
  if (!raw || typeof raw !== "object") return null;
  const o = raw as Record<string, unknown>;
  const id = str(o.id) ?? "";
  const userId = str(o.userId) ?? "";
  const kind = str(o.kind) ?? "";
  if (!id || !userId || !kind) return null;
  const userNameRaw = str(o.userName);
  const userName =
    userNameRaw !== undefined && userNameRaw.trim() !== ""
      ? userNameRaw.trim()
      : undefined;
  const opKindRaw = str(o.opKind);
  const opKind =
    opKindRaw !== undefined && opKindRaw.trim() !== ""
      ? opKindRaw.trim()
      : undefined;
  const layerIds = strList(o.layerIds);
  return {
    id,
    userId,
    ...(userName !== undefined ? { userName } : {}),
    kind,
    ...(opKind !== undefined ? { opKind } : {}),
    sceneRev: num(o.sceneRev),
    ts: num(o.ts),
    ...(str(o.sceneId) ? { sceneId: str(o.sceneId) } : {}),
    ...(str(o.widgetId) ? { widgetId: str(o.widgetId) } : {}),
    ...(str(o.storyId) ? { storyId: str(o.storyId) } : {}),
    ...(str(o.pageId) ? { pageId: str(o.pageId) } : {}),
    ...(str(o.blockId) ? { blockId: str(o.blockId) } : {}),
    ...(str(o.propertyId) ? { propertyId: str(o.propertyId) } : {}),
    ...(str(o.fieldId) ? { fieldId: str(o.fieldId) } : {}),
    ...(str(o.styleId) ? { styleId: str(o.styleId) } : {}),
    ...(str(o.layerId) ? { layerId: str(o.layerId) } : {}),
    ...(layerIds ? { layerIds } : {})
  };
}

/** Normalizes JSON body from GET /api/collab/apply-audit into typed entries. */
export function parseApplyAuditResponse(body: unknown): CollabApplyAuditEntry[] {
  if (!body || typeof body !== "object") return [];
  const entries = (body as { entries?: unknown }).entries;
  if (!Array.isArray(entries)) return [];
  const out: CollabApplyAuditEntry[] = [];
  for (const raw of entries) {
    const e = parseEntry(raw);
    if (e) out.push(e);
  }
  return out;
}
