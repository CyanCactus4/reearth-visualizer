import { Button, Modal, ModalPanel } from "@reearth/app/lib/reearth-ui";
import { useT } from "@reearth/services/i18n/hooks";
import { FC } from "react";

export type CollabLockConflictPayload = {
  resource: string;
  id: string;
  holderUserId: string;
};

type Props = {
  open: boolean;
  payload: CollabLockConflictPayload | null;
  onClose: () => void;
  /** Refetch scene from server (PLAN phase 4 — reconcile without merge UI). */
  onReconcileScene?: () => void;
};

const CollabLockConflictModal: FC<Props> = ({
  open,
  payload,
  onClose,
  onReconcileScene
}) => {
  const t = useT();
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
      </ModalPanel>
    </Modal>
  );
};

export default CollabLockConflictModal;
