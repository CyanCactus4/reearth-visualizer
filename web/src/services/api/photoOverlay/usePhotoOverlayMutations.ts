import { useMutation } from "@apollo/client/react";
import {
  CreateNlsPhotoOverlayInput,
  CreateNlsPhotoOverlayMutation,
  MutationCreateNlsPhotoOverlayArgs,
  MutationRemoveNlsPhotoOverlayArgs,
  RemoveNlsPhotoOverlayInput,
  RemoveNlsPhotoOverlayMutation
} from "@reearth/services/gql";
import {
  CREATE_NLSPHOTOOVERLAY,
  REMOVE_NLSPHOTOOVERLAY
} from "@reearth/services/gql/queries/photoOverlay";
import {
  applyCreateNlsPhotoOverlayPayload,
  applyRemoveNlsPhotoOverlayPayload,
  useCollab
} from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

import { MutationReturn } from "../types";

/** Pass `sceneId` when the editor is bound to a scene so photo overlay toggles use collab WS. */
export const usePhotoOverlayMutations = (sceneIdForCollab?: string) => {
  const t = useT();
  const [, setNotification] = useNotification();
  const collab = useCollab();

  const [createNLSPhotoOverlayMutation] = useMutation<
    CreateNlsPhotoOverlayMutation,
    MutationCreateNlsPhotoOverlayArgs
  >(CREATE_NLSPHOTOOVERLAY, { refetchQueries: ["GetScene"] });

  const [removeNLSPhotoOverlayMutation] = useMutation<
    RemoveNlsPhotoOverlayMutation,
    MutationRemoveNlsPhotoOverlayArgs
  >(REMOVE_NLSPHOTOOVERLAY, { refetchQueries: ["GetScene"] });

  const createNLSPhotoOverlay = useCallback(
    async (
      input: CreateNlsPhotoOverlayInput
    ): Promise<MutationReturn<CreateNlsPhotoOverlayMutation>> => {
      if (collab && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyCreateNlsPhotoOverlayPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully enabled photo overlay")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await createNLSPhotoOverlayMutation({
        variables: { input }
      });
      if (error || !data?.createNLSPhotoOverlay?.layer?.id) {
        setNotification({ type: "error", text: t("Failed to add layer.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully enabled photo overlay")
      });

      return { data, status: "success" };
    },
    [collab, createNLSPhotoOverlayMutation, sceneIdForCollab, setNotification, t]
  );

  const removeNLSPhotoOverlay = useCallback(
    async (
      input: RemoveNlsPhotoOverlayInput
    ): Promise<MutationReturn<RemoveNlsPhotoOverlayMutation>> => {
      if (collab && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyRemoveNlsPhotoOverlayPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "info",
            text: t("Photo overlay removed from layer")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await removeNLSPhotoOverlayMutation({
        variables: { input }
      });
      if (error || !data?.removeNLSPhotoOverlay?.layer?.id) {
        setNotification({
          type: "error",
          text: t("Failed to remove photo overlay from layer")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Photo overlay removed from layer")
      });

      return { data, status: "success" };
    },
    [collab, removeNLSPhotoOverlayMutation, sceneIdForCollab, setNotification, t]
  );

  return {
    createNLSPhotoOverlay,
    removeNLSPhotoOverlay
  };
};
