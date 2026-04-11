/** Minimal widget row from GetScene `node` for merge-compare (no property fragments). */
export type SceneMergeWidgetRow = {
  id: string;
  enabled?: boolean | null;
  extended?: boolean | null;
  pluginId?: string | null;
  extensionId?: string | null;
};

export type SceneMergeStorySummary = {
  storyIds: string[];
  pageCount: number;
};

export type SceneMergeRichDiff = {
  widgetDiff: {
    added: string[];
    removed: string[];
    changed: { id: string; fields: string[] }[];
  };
  storySummary: {
    cacheStories: number;
    networkStories: number;
    cachePages: number;
    networkPages: number;
  };
};

function widgetsFromSceneNode(node: unknown): SceneMergeWidgetRow[] {
  const n = node as { widgets?: unknown } | null | undefined;
  const w = n?.widgets;
  if (!Array.isArray(w)) return [];
  const out: SceneMergeWidgetRow[] = [];
  for (const x of w) {
    const o = x as Record<string, unknown>;
    if (typeof o?.id !== "string") continue;
    out.push({
      id: o.id,
      enabled: typeof o.enabled === "boolean" ? o.enabled : null,
      extended: typeof o.extended === "boolean" ? o.extended : null,
      pluginId: typeof o.pluginId === "string" ? o.pluginId : null,
      extensionId: typeof o.extensionId === "string" ? o.extensionId : null
    });
  }
  return out;
}

function storySummaryFromSceneNode(node: unknown): SceneMergeStorySummary {
  const n = node as { stories?: unknown } | null | undefined;
  const stories = n?.stories;
  if (!Array.isArray(stories)) {
    return { storyIds: [], pageCount: 0 };
  }
  const storyIds: string[] = [];
  let pageCount = 0;
  for (const s of stories) {
    const o = s as Record<string, unknown>;
    if (typeof o?.id === "string") storyIds.push(o.id);
    const pages = o?.pages;
    if (Array.isArray(pages)) pageCount += pages.length;
  }
  return { storyIds, pageCount };
}

function widgetSignature(w: SceneMergeWidgetRow): string {
  return JSON.stringify({
    id: w.id,
    e: w.enabled ?? null,
    x: w.extended ?? null,
    p: w.pluginId ?? "",
    ex: w.extensionId ?? ""
  });
}

/** Compare two GetScene payloads for a richer lock-conflict / merge preview. */
export function sceneMergeRichDiff(
  cacheData: unknown,
  networkData: unknown
): SceneMergeRichDiff | null {
  const cNode = (cacheData as { node?: unknown } | undefined)?.node;
  const nNode = (networkData as { node?: unknown } | undefined)?.node;
  if (
    !cNode ||
    !nNode ||
    (cNode as { __typename?: string }).__typename !== "Scene" ||
    (nNode as { __typename?: string }).__typename !== "Scene"
  ) {
    return null;
  }
  const cw = widgetsFromSceneNode(cNode);
  const nw = widgetsFromSceneNode(nNode);
  const cmap = new Map(cw.map((w) => [w.id, w]));
  const nmap = new Map(nw.map((w) => [w.id, w]));
  const added: string[] = [];
  const removed: string[] = [];
  const changed: { id: string; fields: string[] }[] = [];
  for (const [id, w] of nmap) {
    if (!cmap.has(id)) added.push(id);
    else if (widgetSignature(w) !== widgetSignature(cmap.get(id)!)) {
      const prev = cmap.get(id)!;
      const fields: string[] = [];
      if (prev.enabled !== w.enabled) fields.push("enabled");
      if (prev.extended !== w.extended) fields.push("extended");
      if (prev.pluginId !== w.pluginId) fields.push("pluginId");
      if (prev.extensionId !== w.extensionId) fields.push("extensionId");
      changed.push({ id, fields: fields.length ? fields : ["widget"] });
    }
  }
  for (const id of cmap.keys()) {
    if (!nmap.has(id)) removed.push(id);
  }
  const cs = storySummaryFromSceneNode(cNode);
  const ns = storySummaryFromSceneNode(nNode);
  return {
    widgetDiff: { added, removed, changed },
    storySummary: {
      cacheStories: cs.storyIds.length,
      networkStories: ns.storyIds.length,
      cachePages: cs.pageCount,
      networkPages: ns.pageCount
    }
  };
}
