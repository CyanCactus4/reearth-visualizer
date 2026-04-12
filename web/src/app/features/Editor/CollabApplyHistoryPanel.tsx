import { useApolloClient } from "@apollo/client/react";
import { useAuth } from "@reearth/services/auth/useAuth";
import { useMe } from "@reearth/services/api/user/useMeQueries";
import {
  buildCollabApplyAuditUrl,
  collabAuditEntryActionLabel,
  collabPeerAppliedTargetHint,
  formatCollabAuditTimestamp,
  parseApplyAuditResponse,
  postCollabAdminRestore,
  postCollabRedo,
  postCollabUndo,
  useCollab,
  type CollabApplyAuditEntry
} from "@reearth/services/collab";
import { GET_SCENE } from "@reearth/services/gql/queries/scene";
import { useLang, useT } from "@reearth/services/i18n/hooks";
import {
  type CSSProperties,
  FC,
  useCallback,
  useEffect,
  useMemo,
  useState
} from "react";

type Props = { sceneId: string };

const btnBase: CSSProperties = {
  fontSize: 12,
  background: "rgba(255,255,255,0.1)",
  border: "1px solid rgba(255,255,255,0.22)",
  color: "inherit",
  borderRadius: 6,
  cursor: "pointer",
  padding: "6px 12px",
  fontWeight: 500
};

const btnMuted: CSSProperties = {
  ...btnBase,
  background: "rgba(255,255,255,0.06)",
  opacity: 0.85
};

/** Read-only collab apply journal + server undo/redo when configured. */
const CollabApplyHistoryPanel: FC<Props> = ({ sceneId }) => {
  const collab = useCollab();
  const t = useT();
  const lang = useLang();
  const { me } = useMe({ skip: !collab?.projectId });
  const apollo = useApolloClient();
  const { getAccessToken } = useAuth();
  const [entries, setEntries] = useState<CollabApplyAuditEntry[]>([]);
  const [journalOpen, setJournalOpen] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [undoMsg, setUndoMsg] = useState<string | null>(null);
  const [restoreMsg, setRestoreMsg] = useState<string | null>(null);
  const [selectedRestoreRev, setSelectedRestoreRev] = useState<string>("");

  const liveOpen = collab?.status === "open";

  const load = useCallback(async () => {
    if (!collab?.projectId) return;
    setErr(null);
    setLoading(true);
    try {
      const token = await getAccessToken();
      const apiBase = window.REEARTH_CONFIG?.api || "/api";
      const url = buildCollabApplyAuditUrl(
        apiBase,
        collab.projectId,
        100,
        sceneId
      );
      const res = await fetch(url, {
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      });
      if (!res.ok) {
        setErr(t("Collab history load failed"));
        setEntries([]);
        return;
      }
      const body: unknown = await res.json();
      setEntries(parseApplyAuditResponse(body));
    } catch {
      setErr(t("Collab history load failed"));
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, [collab?.projectId, getAccessToken, sceneId, t]);

  const restoreRevOptions = useMemo(() => {
    const s = new Set<number>();
    for (const e of entries) {
      if (e.sceneRev > 0) s.add(e.sceneRev);
    }
    return Array.from(s).sort((a, b) => b - a);
  }, [entries]);

  useEffect(() => {
    if (!journalOpen) return;
    if (restoreRevOptions.length === 0) return;
    if (
      selectedRestoreRev &&
      restoreRevOptions.includes(Number(selectedRestoreRev))
    ) {
      return;
    }
    setSelectedRestoreRev(String(restoreRevOptions[0]));
  }, [journalOpen, restoreRevOptions, selectedRestoreRev]);

  useEffect(() => {
    if (!collab?.projectId) return;
    void load();
  }, [collab?.projectId, sceneId, load]);

  useEffect(() => {
    if (collab?.lastMessage?.t !== "applied") return;
    const id = setTimeout(() => void load(), 450);
    return () => clearTimeout(id);
  }, [collab?.lastMessage, load]);

  const runUndo = useCallback(async () => {
    setUndoMsg(null);
    const apiBase = window.REEARTH_CONFIG?.api || "/api";
    try {
      const res = await postCollabUndo(
        apiBase,
        getAccessToken,
        sceneId,
        collab?.collabReplicaId
      );
      if (!res.ok) {
        setUndoMsg(t("Collab undo failed"));
        return;
      }
      setUndoMsg(t("Collab undo ok"));
      void apollo.query({
        query: GET_SCENE,
        variables: { sceneId, lang },
        fetchPolicy: "network-only"
      });
      void load();
    } catch {
      setUndoMsg(t("Collab undo failed"));
    }
  }, [
    apollo,
    collab?.collabReplicaId,
    getAccessToken,
    lang,
    load,
    sceneId,
    t
  ]);

  const runRedo = useCallback(async () => {
    setUndoMsg(null);
    const apiBase = window.REEARTH_CONFIG?.api || "/api";
    try {
      const res = await postCollabRedo(
        apiBase,
        getAccessToken,
        sceneId,
        collab?.collabReplicaId
      );
      if (!res.ok) {
        setUndoMsg(t("Collab redo failed"));
        return;
      }
      setUndoMsg(t("Collab redo ok"));
      void apollo.query({
        query: GET_SCENE,
        variables: { sceneId, lang },
        fetchPolicy: "network-only"
      });
      void load();
    } catch {
      setUndoMsg(t("Collab redo failed"));
    }
  }, [
    apollo,
    collab?.collabReplicaId,
    getAccessToken,
    lang,
    load,
    sceneId,
    t
  ]);

  const runAdminRestore = useCallback(async () => {
    setRestoreMsg(null);
    const rev = Number(selectedRestoreRev);
    if (!Number.isFinite(rev) || rev <= 0) {
      setRestoreMsg(t("Collab admin restore not found"));
      return;
    }
    if (!window.confirm(t("Collab admin restore confirm"))) return;
    const apiBase = window.REEARTH_CONFIG?.api || "/api";
    try {
      const res = await postCollabAdminRestore(
        apiBase,
        getAccessToken,
        sceneId,
        rev
      );
      if (res.status === 200) {
        setRestoreMsg(t("Collab admin restore ok"));
        void apollo.query({
          query: GET_SCENE,
          variables: { sceneId, lang },
          fetchPolicy: "network-only"
        });
        void load();
        return;
      }
      if (res.status === 403) {
        setRestoreMsg(t("Collab admin restore forbidden"));
        return;
      }
      if (res.status === 501) {
        setRestoreMsg(t("Collab admin restore no snapshots"));
        return;
      }
      if (res.status === 404) {
        setRestoreMsg(t("Collab admin restore not found"));
        return;
      }
      setRestoreMsg(t("Collab admin restore failed"));
    } catch {
      setRestoreMsg(t("Collab admin restore failed"));
    }
  }, [
    apollo,
    getAccessToken,
    lang,
    load,
    sceneId,
    selectedRestoreRev,
    t
  ]);

  if (!collab?.projectId) return null;

  const undoTitle = liveOpen ? undefined : t("Collab history undo needs live");

  return (
    <div
      data-testid="collab-apply-history"
      style={{
        fontSize: 12,
        padding: "8px 10px 10px",
        borderBottom: "1px solid rgba(255,255,255,0.08)"
      }}
    >
      <div
        style={{
          fontSize: 11,
          opacity: 0.82,
          lineHeight: 1.4,
          marginBottom: 8
        }}
      >
        {t("Collab history location hint")}
      </div>

      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: 8,
          alignItems: "center",
          marginBottom: 8
        }}
      >
        <button
          type="button"
          onClick={() => void runUndo()}
          disabled={!liveOpen}
          title={undoTitle}
          style={{
            ...btnBase,
            ...(!liveOpen ? { opacity: 0.45, cursor: "not-allowed" } : {})
          }}
        >
          {t("Collab undo")}
        </button>
        <button
          type="button"
          onClick={() => void runRedo()}
          disabled={!liveOpen}
          title={undoTitle}
          style={{
            ...btnBase,
            ...(!liveOpen ? { opacity: 0.45, cursor: "not-allowed" } : {})
          }}
        >
          {t("Collab redo")}
        </button>
        <button
          type="button"
          onClick={() => void load()}
          disabled={loading}
          style={{ ...btnMuted, cursor: loading ? "wait" : "pointer" }}
        >
          {loading ? t("Collab history loading") : t("Collab history refresh")}
        </button>
        {entries.length > 0 ? (
          <span
            style={{
              fontSize: 11,
              opacity: 0.75,
              marginLeft: 4
            }}
          >
            {t("Collab history count badge", {
              count: entries.length
            })}
          </span>
        ) : null}
        <button
          type="button"
          onClick={() => setJournalOpen((o) => !o)}
          style={{
            marginLeft: "auto",
            fontSize: 12,
            background: "transparent",
            border: "none",
            color: "inherit",
            cursor: "pointer",
            textDecoration: "underline",
            opacity: 0.9
          }}
        >
          {journalOpen
            ? t("Collab history journal hide")
            : t("Collab history journal show")}
        </button>
      </div>

      {!liveOpen ? (
        <div style={{ fontSize: 11, opacity: 0.75, marginBottom: 6 }}>
          {t("Collab history undo needs live")}
        </div>
      ) : null}

      {undoMsg ? (
        <div style={{ fontSize: 12, marginBottom: 6, opacity: 0.9 }}>
          {undoMsg}
        </div>
      ) : null}

      {journalOpen ? (
        <div
          style={{
            marginTop: 4,
            maxHeight: "min(42vh, 380px)",
            overflowY: "auto",
            paddingRight: 4
          }}
        >
          <div
            style={{
              fontSize: 11,
              opacity: 0.82,
              marginBottom: 10,
              lineHeight: 1.45
            }}
          >
            {t("Collab history undo scope note")}
          </div>

          <div
            style={{
              marginTop: 8,
              marginBottom: 12,
              paddingTop: 10,
              paddingBottom: 10,
              borderTop: "1px solid rgba(255,255,255,0.1)",
              borderBottom: "1px solid rgba(255,255,255,0.08)"
            }}
          >
            <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 6 }}>
              {t("Collab admin restore heading")}
            </div>
            <div
              style={{
                display: "flex",
                flexWrap: "wrap",
                gap: 8,
                alignItems: "center"
              }}
            >
              <label
                style={{
                  fontSize: 11,
                  display: "flex",
                  gap: 6,
                  alignItems: "center"
                }}
              >
                <span style={{ opacity: 0.85 }}>
                  {t("Collab admin restore pick rev")}
                </span>
                <select
                  value={selectedRestoreRev}
                  onChange={(e) => setSelectedRestoreRev(e.target.value)}
                  style={{
                    fontSize: 12,
                    minHeight: 28,
                    maxWidth: 220,
                    background: "rgba(0,0,0,0.28)",
                    color: "inherit",
                    border: "1px solid rgba(255,255,255,0.22)",
                    borderRadius: 6
                  }}
                >
                  {restoreRevOptions.length === 0 ? (
                    <option value="">{t("Collab history empty")}</option>
                  ) : (
                    restoreRevOptions.map((r) => (
                      <option key={r} value={String(r)}>
                        rev {r}
                      </option>
                    ))
                  )}
                </select>
              </label>
              <button
                type="button"
                onClick={() => void runAdminRestore()}
                disabled={restoreRevOptions.length === 0}
                style={{
                  fontSize: 12,
                  background: "rgba(200,120,80,0.28)",
                  border: "1px solid rgba(255,200,160,0.38)",
                  color: "inherit",
                  borderRadius: 6,
                  cursor:
                    restoreRevOptions.length === 0 ? "not-allowed" : "pointer",
                  padding: "5px 12px"
                }}
              >
                {t("Collab admin restore run")}
              </button>
            </div>
            {restoreMsg ? (
              <div style={{ fontSize: 12, marginTop: 6, opacity: 0.9 }}>
                {restoreMsg}
              </div>
            ) : null}
          </div>

          {err ? (
            <div style={{ color: "#f88", fontSize: 12, marginBottom: 8 }}>
              {err}
            </div>
          ) : loading && entries.length === 0 ? (
            <div style={{ opacity: 0.75, fontSize: 12 }}>
              {t("Collab history loading")}
            </div>
          ) : entries.length === 0 ? (
            <div style={{ opacity: 0.75, fontSize: 12 }}>
              {t("Collab history empty")}
            </div>
          ) : (
            <ul
              style={{
                margin: "4px 0 0",
                padding: 0,
                listStyle: "none",
                display: "flex",
                flexDirection: "column",
                gap: 8
              }}
            >
              {entries.map((e) => {
                const isYou = me?.id && e.userId === me.id;
                const action = collabAuditEntryActionLabel(
                  e.kind,
                  e.opKind,
                  t
                );
                const target = collabPeerAppliedTargetHint({
                  widgetId: e.widgetId,
                  storyId: e.storyId,
                  pageId: e.pageId,
                  blockId: e.blockId,
                  propertyId: e.propertyId,
                  fieldId: e.fieldId,
                  styleId: e.styleId,
                  layerId: e.layerId,
                  layerIds: e.layerIds
                });
                const when = formatCollabAuditTimestamp(e.ts, lang);
                const authorLabel =
                  me?.id && e.userId === me.id
                    ? t("Collab history author you")
                    : e.userName?.trim()
                      ? e.userName.trim()
                      : e.userId.length > 10
                        ? `${e.userId.slice(0, 8)}…`
                        : e.userId;
                const detailBits = [
                  isYou
                    ? t("Collab history entry you")
                    : t("Collab history entry other"),
                  authorLabel,
                  `rev ${e.sceneRev}`,
                  target || null
                ].filter(Boolean);

                return (
                  <li
                    key={e.id}
                    style={{
                      padding: "10px 12px",
                      borderRadius: 8,
                      background: "rgba(0,0,0,0.22)",
                      borderLeft: `4px solid ${
                        isYou
                          ? "rgba(100,200,140,0.9)"
                          : "rgba(120,160,255,0.85)"
                      }`
                    }}
                  >
                    <div
                      style={{
                        fontSize: 12,
                        fontWeight: 600,
                        lineHeight: 1.35,
                        marginBottom: 4
                      }}
                    >
                      <span style={{ opacity: 0.8, fontWeight: 500 }}>
                        {when}
                      </span>
                      <span style={{ opacity: 0.45, margin: "0 6px" }}>·</span>
                      <span>{action}</span>
                    </div>
                    <div
                      style={{
                        fontSize: 11,
                        opacity: 0.82,
                        lineHeight: 1.4,
                        wordBreak: "break-word"
                      }}
                      title={e.userId + (e.userName ? ` (${e.userName})` : "")}
                    >
                      {detailBits.join(" · ")}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      ) : null}
    </div>
  );
};

export default CollabApplyHistoryPanel;
