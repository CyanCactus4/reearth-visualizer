/** Short id for toast detail lines (not security-sensitive). */
function shortId(id: string): string {
  const t = id.trim();
  if (t.length <= 10) return t;
  return `${t.slice(0, 8)}…`;
}

type AppliedPayload = {
  kind?: string;
  opKind?: string;
  widgetId?: string;
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

function humanizeKind(domainKind: string): string {
  return domainKind.replace(/_/g, " ");
}

/** i18n label for a domain apply kind (or opKind); falls back to humanized id. */
export function collabApplyKindLabel(
  domainKind: string,
  t: (key: string) => string
): string {
  const k = domainKind.trim();
  if (!k) return "";
  const key = `Collab apply kind ${k}`;
  const s = t(key);
  if (s !== key) return s;
  return humanizeKind(k);
}

/** Human-readable primary line for one apply-audit row (journal UI). */
export function collabAuditEntryActionLabel(
  kind: string,
  opKind: string | undefined,
  t: (key: string, opts?: Record<string, string>) => string
): string {
  const ok = (opKind ?? "").trim();
  if (kind === "collab_undo") {
    return t("Collab audit line undo", {
      op: collabApplyKindLabel(ok || "edit", (k) => t(k))
    });
  }
  if (kind === "collab_redo") {
    return t("Collab audit line redo", {
      op: collabApplyKindLabel(ok || "edit", (k) => t(k))
    });
  }
  return collabApplyKindLabel(kind || "edit", (k) => t(k));
}

/** Format server audit `ts` (unix seconds or ms). */
export function formatCollabAuditTimestamp(ts: number, locale: string): string {
  if (!ts || ts < 0) return "—";
  const ms = ts < 1_000_000_000_000 ? ts * 1000 : ts;
  try {
    return new Intl.DateTimeFormat(locale || "en", {
      dateStyle: "short",
      timeStyle: "medium"
    }).format(new Date(ms));
  } catch {
    return String(ts);
  }
}

/** Action phrase for a peer `applied` message (undo/redo vs normal apply). */
export function collabPeerAppliedAction(
  kind: string,
  opKind: string,
  t: (key: string, opts?: Record<string, string>) => string
): string {
  const ok = opKind.trim();
  if (kind === "collab_undo") {
    return t("Collab apply meta undo", { op: collabApplyKindLabel(ok || "edit", t) });
  }
  if (kind === "collab_redo") {
    return t("Collab apply meta redo", { op: collabApplyKindLabel(ok || "edit", t) });
  }
  return collabApplyKindLabel(kind || "edit", t);
}

/** Optional target hint (ids) appended to the toast. */
export function collabPeerAppliedTargetHint(d: AppliedPayload): string {
  const parts: string[] = [];
  if (Array.isArray(d.layerIds) && d.layerIds.length > 0) {
    parts.push(`${d.layerIds.length} layers`);
  } else if (typeof d.layerId === "string" && d.layerId) {
    parts.push(`layer ${shortId(d.layerId)}`);
  }
  if (typeof d.widgetId === "string" && d.widgetId) {
    parts.push(`widget ${shortId(d.widgetId)}`);
  }
  if (typeof d.blockId === "string" && d.blockId) {
    parts.push(`block ${shortId(d.blockId)}`);
  }
  if (typeof d.styleId === "string" && d.styleId) {
    parts.push(`style ${shortId(d.styleId)}`);
  }
  if (typeof d.propertyId === "string" && d.propertyId) {
    const f =
      typeof d.fieldId === "string" && d.fieldId ? ` / ${shortId(d.fieldId)}` : "";
    parts.push(`property ${shortId(d.propertyId)}${f}`);
  }
  if (typeof d.itemId === "string" && d.itemId) {
    parts.push(`item ${shortId(d.itemId)}`);
  }
  if (typeof d.pageId === "string" && d.pageId) {
    parts.push(`page ${shortId(d.pageId)}`);
  }
  if (typeof d.storyId === "string" && d.storyId) {
    parts.push(`story ${shortId(d.storyId)}`);
  }
  return parts.join(", ");
}
