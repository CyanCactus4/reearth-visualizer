import { useMutation } from "@apollo/client/react";
import {
  CreateNlsInfoboxInput,
  CreateNlsInfoboxMutation,
  MutationCreateNlsInfoboxArgs,
  MutationRemoveNlsInfoboxArgs,
  RemoveNlsInfoboxInput,
  RemoveNlsInfoboxMutation
} from "@reearth/services/gql";
import {
  CREATE_NLSINFOBOX,
  REMOVE_NLSINFOBOX
} from "@reearth/services/gql/queries/infobox";
import {
  applyCreateNlsInfoboxPayload,
  applyRemoveNlsInfoboxPayload,
  useCollab
} from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

import { MutationReturn } from "../types";

/** Pass `sceneId` when the editor is bound to a scene so infobox enable/disable uses collab WS. */
export const useInfoboxMutations = (sceneIdForCollab?: string) => {
  const t = useT();
  const [, setNotification] = useNotification();
  const collab = useCollab();

  const [createNLSInfoboxMutation] = useMutation<
    CreateNlsInfoboxMutation,
    MutationCreateNlsInfoboxArgs
  >(CREATE_NLSINFOBOX, { refetchQueries: ["GetScene"] });

  const [removeNLSInfoboxMutation] = useMutation<
    RemoveNlsInfoboxMutation,
    MutationRemoveNlsInfoboxArgs
  >(REMOVE_NLSINFOBOX, { refetchQueries: ["GetScene"] });

  const createNLSInfobox = useCallback(
    async (
      input: CreateNlsInfoboxInput
    ): Promise<MutationReturn<CreateNlsInfoboxMutation>> => {
      if (collab?.status === "open" && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyCreateNlsInfoboxPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully added a new layer")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await createNLSInfoboxMutation({
        variables: { input }
      });
      if (error || !data?.createNLSInfobox?.layer?.id) {
        setNotification({ type: "error", text: t("Failed to add layer.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully added a new layer")
      });

      return { data, status: "success" };
    },
    [collab, createNLSInfoboxMutation, sceneIdForCollab, setNotification, t]
  );

  const removeNLSInfobox = useCallback(
    async (
      input: RemoveNlsInfoboxInput
    ): Promise<MutationReturn<RemoveNlsInfoboxMutation>> => {
      if (collab?.status === "open" && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyRemoveNlsInfoboxPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "info",
            text: t("Infobox removed from layer")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await removeNLSInfoboxMutation({
        variables: { input }
      });
      if (error || !data?.removeNLSInfobox?.layer?.id) {
        setNotification({
          type: "error",
          text: t("Failed to remove infobox from layer")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Infobox removed from layer")
      });

      return { data, status: "success" };
    },
    [collab, removeNLSInfoboxMutation, sceneIdForCollab, setNotification, t]
  );

  return {
    createNLSInfobox,
    removeNLSInfobox
  };
};
