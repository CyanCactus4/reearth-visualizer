import type { CollabChatLine } from "@reearth/services/collab";
import { useCollab } from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { type FC, type ReactNode, useCallback, useEffect, useRef, useState } from "react";

const mentionSplit = /@([a-zA-Z0-9_-]+)/g;

function ChatLineText({
  text,
  mentions
}: Pick<CollabChatLine, "text" | "mentions">) {
  const mentionSet = new Set(mentions ?? []);
  const parts: ReactNode[] = [];
  let last = 0;
  let partKey = 0;
  mentionSplit.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = mentionSplit.exec(text)) !== null) {
    if (m.index > last) {
      parts.push(text.slice(last, m.index));
    }
    const handle = m[1] ?? "";
    const full = m[0];
    const isKnown = mentionSet.has(handle);
    partKey += 1;
    parts.push(
      <span
        key={`m${partKey}`}
        style={{
          color: isKnown ? "rgba(120,200,255,0.95)" : "rgba(255,255,255,0.75)",
          fontWeight: isKnown ? 600 : 400
        }}
      >
        {full}
      </span>
    );
    last = m.index + full.length;
  }
  if (last < text.length) {
    parts.push(text.slice(last));
  }
  return <>{parts.length ? parts : text}</>;
}

const CollabChatPanel: FC = () => {
  const collab = useCollab();
  const t = useT();
  const [draft, setDraft] = useState("");
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = endRef.current;
    if (el && typeof el.scrollIntoView === "function") {
      el.scrollIntoView({ behavior: "smooth", block: "end" });
    }
  }, [collab?.chatMessages.length]);

  const onSend = useCallback(() => {
    if (!collab || collab.status !== "open") return;
    const line = draft.trim();
    if (!line) return;
    collab.sendChat(line);
    setDraft("");
  }, [collab, draft]);

  if (!collab?.projectId) {
    return null;
  }

  return (
    <div
      data-testid="collab-chat-panel"
      style={{
        fontSize: 11,
        lineHeight: 1.4,
        padding: "4px 8px",
        borderBottom: "1px solid rgba(255,255,255,0.08)"
      }}
    >
      <div
        style={{
          maxHeight: 120,
          overflowY: "auto",
          marginBottom: 4,
          opacity: 0.9
        }}
      >
        {collab.chatMessages.map((m) => (
          <div
            key={m.id}
            data-testid={`collab-chat-line-${m.id}`}
            style={{ opacity: m.pending ? 0.55 : 0.9 }}
          >
            <strong style={{ fontWeight: 600 }}>{m.userId}</strong>:{" "}
            <ChatLineText text={m.text} mentions={m.mentions} />
            {m.pending ? " …" : null}
          </div>
        ))}
        <div ref={endRef} />
      </div>
      <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
        <input
          type="text"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder={t("Collab chat placeholder")}
          disabled={collab.status !== "open"}
          style={{
            flex: 1,
            fontSize: 11,
            padding: "2px 6px",
            minWidth: 0
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              onSend();
            }
          }}
        />
        <button
          type="button"
          onClick={onSend}
          disabled={collab.status !== "open" || !draft.trim()}
          style={{ fontSize: 11, whiteSpace: "nowrap" }}
        >
          {t("Collab chat send")}
        </button>
      </div>
    </div>
  );
};

export default CollabChatPanel;
