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
};

const CollabLockConflictModal: FC<Props> = ({ open, payload, onClose }) => {
  const t = useT();
  if (!open || !payload) return null;

  return (
    <Modal visible size="small" data-testid="collab-lock-conflict-modal">
      <ModalPanel
        title={t("Collab edit conflict")}
        onCancel={onClose}
        actions={
          <Button title={t("OK")} onClick={onClose} data-testid="collab-lock-conflict-ok" />
        }
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
