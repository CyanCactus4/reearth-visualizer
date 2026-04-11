import { Button, Modal, ModalPanel } from "@reearth/app/lib/reearth-ui";
import { useT } from "@reearth/services/i18n/hooks";
import { FC, useCallback, useEffect, useState } from "react";

export type CollabLockConflictPayload = {
  resource: string;
  id: string;
  holderUserId: string;
};

export type CollabLockConflictSnapshots = {
  cache: { widgets: number; stories: number };
  network: { widgets: number; stories: number };
};

type Props = {
  open: boolean;
  payload: CollabLockConflictPayload | null;
  onClose: () => void;
  /** Refetch scene from server (PLAN phase 4 — reconcile without merge UI). */
  onReconcileScene?: () => void;
  /** Load Apollo cache vs network summaries for a lightweight two-snapshot compare. */
  onCompareSnapshots?: () => Promise<CollabLockConflictSnapshots | null>;
};

const CollabLockConflictModal: FC<Props> = ({
  open,
  payload,
  onClose,
  onReconcileScene,
  onCompareSnapshots
}) => {
  const t = useT();
  const [snap, setSnap] = useState<CollabLockConflictSnapshots | null>(null);
  const [compareErr, setCompareErr] = useState<string | null>(null);
  const [loadingCompare, setLoadingCompare] = useState(false);

  useEffect(() => {
    if (!open) {
      setSnap(null);
      setCompareErr(null);
    }
  }, [open]);

  const runCompare = useCallback(async () => {
    if (!onCompareSnapshots) return;
    setCompareErr(null);
    setLoadingCompare(true);
    try {
      const r = await onCompareSnapshots();
      setSnap(r);
      if (!r) setCompareErr(t("Collab lock conflict compare failed"));
    } catch {
      setCompareErr(t("Collab lock conflict compare failed"));
    } finally {
      setLoadingCompare(false);
    }
  }, [onCompareSnapshots, t]);

  if (!open || !payload) return null;

  const actions =
    onReconcileScene != null
      ? [
          <Button
            key="reload"
            title={t("Collab lock conflict reload scene")}
            appearance="secondary"
            onClick={() => {
              onReconcileScene();
              onClose();
            }}
            data-testid="collab-lock-conflict-reload"
          />,
          <Button
            key="ok"
            title={t("OK")}
            onClick={onClose}
            data-testid="collab-lock-conflict-ok"
          />
        ]
      : [
          <Button
            key="ok"
            title={t("OK")}
            onClick={onClose}
            data-testid="collab-lock-conflict-ok"
          />
        ];

  return (
    <Modal visible size="small" data-testid="collab-lock-conflict-modal">
      <ModalPanel
        title={t("Collab edit conflict")}
        onCancel={onClose}
        actions={actions}
      >
        <p style={{ margin: 0, fontSize: 13, lineHeight: 1.45 }}>
          {t("Collab lock conflict description", {
            resource: payload.resource,
            id: payload.id,
            userId: payload.holderUserId
          })}
        </p>
        {onCompareSnapshots ? (
          <div style={{ marginTop: 12 }}>
            <Button
              title={t("Collab lock conflict compare")}
              appearance="simple"
              onClick={() => void runCompare()}
              disabled={loadingCompare}
              data-testid="collab-lock-conflict-compare"
            />
            {compareErr ? (
              <p style={{ margin: "8px 0 0", fontSize: 11, color: "#f88" }}>
                {compareErr}
              </p>
            ) : null}
            {snap ? (
              <table
                style={{
                  width: "100%",
                  marginTop: 8,
                  fontSize: 11,
                  borderCollapse: "collapse"
                }}
              >
                <thead>
                  <tr>
                    <th style={{ textAlign: "left", padding: 4 }} />
                    <th style={{ textAlign: "left", padding: 4 }}>
                      {t("Collab lock conflict col cache")}
                    </th>
                    <th style={{ textAlign: "left", padding: 4 }}>
                      {t("Collab lock conflict col network")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td style={{ padding: 4 }}>widgets</td>
                    <td style={{ padding: 4 }}>{snap.cache.widgets}</td>
                    <td style={{ padding: 4 }}>{snap.network.widgets}</td>
                  </tr>
                  <tr>
                    <td style={{ padding: 4 }}>stories</td>
                    <td style={{ padding: 4 }}>{snap.cache.stories}</td>
                    <td style={{ padding: 4 }}>{snap.network.stories}</td>
                  </tr>
                </tbody>
              </table>
            ) : null}
          </div>
        ) : null}
      </ModalPanel>
    </Modal>
  );
};

export default CollabLockConflictModal;
