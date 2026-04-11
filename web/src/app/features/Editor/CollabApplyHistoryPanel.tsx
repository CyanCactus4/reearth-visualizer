import { useAuth } from "@reearth/services/auth/useAuth";
import {
  buildCollabApplyAuditUrl,
  useCollab
} from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { FC, useCallback, useEffect, useState } from "react";

type Entry = {
  id: string;
  userId: string;
  kind: string;
  sceneRev: number;
  ts: number;
  widgetId?: string;
  storyId?: string;
  pageId?: string;
  blockId?: string;
};

/** Read-only collab apply journal (PLAN phase 6 UI slice). */
const CollabApplyHistoryPanel: FC = () => {
  const collab = useCollab();
  const t = useT();
  const { getAccessToken } = useAuth();
  const [entries, setEntries] = useState<Entry[]>([]);
  const [open, setOpen] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!collab?.projectId) return;
    setErr(null);
    try {
      const token = await getAccessToken();
      const apiBase = window.REEARTH_CONFIG?.api || "/api";
      const url = buildCollabApplyAuditUrl(apiBase, collab.projectId, 50);
      const res = await fetch(url, {
        credentials: "include",
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      });
      if (!res.ok) {
        setErr(t("Collab history load failed"));
        setEntries([]);
        return;
      }
      const body = (await res.json()) as { entries?: Entry[] };
      setEntries(Array.isArray(body.entries) ? body.entries : []);
    } catch {
      setErr(t("Collab history load failed"));
      setEntries([]);
    }
  }, [collab?.projectId, getAccessToken, t]);

  useEffect(() => {
    if (open) void load();
  }, [open, load]);

  if (!collab?.projectId) return null;

  return (
    <div
      data-testid="collab-apply-history"
      style={{
        fontSize: 11,
        padding: "2px 8px 6px",
        borderBottom: "1px solid rgba(255,255,255,0.08)"
      }}
    >
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        style={{
          fontSize: 11,
          background: "transparent",
          border: "none",
          color: "inherit",
          cursor: "pointer",
          textDecoration: "underline"
        }}
      >
        {open
          ? t("Collab history hide")
          : t("Collab history show")}
      </button>
      {open ? (
        <div style={{ marginTop: 6, maxHeight: 140, overflowY: "auto" }}>
          {err ? (
            <div style={{ color: "#f88" }}>{err}</div>
          ) : entries.length === 0 ? (
            <div style={{ opacity: 0.7 }}>{t("Collab history empty")}</div>
          ) : (
            <ul style={{ margin: 0, paddingLeft: 16 }}>
              {entries.map((e) => (
                <li key={e.id} style={{ marginBottom: 4 }}>
                  <code>{e.kind}</code> · rev {e.sceneRev} ·{" "}
                  <span style={{ opacity: 0.85 }}>{e.userId}</span>
                  {e.widgetId ? ` · w ${e.widgetId.slice(0, 8)}…` : null}
                  {e.storyId ? ` · story` : null}
                </li>
              ))}
            </ul>
          )}
        </div>
      ) : null}
    </div>
  );
};

export default CollabApplyHistoryPanel;
