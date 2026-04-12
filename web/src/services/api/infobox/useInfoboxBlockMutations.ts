import { useMutation } from "@apollo/client/react";
import {
  AddNlsInfoboxBlockInput,
  AddNlsInfoboxBlockMutation,
  MoveNlsInfoboxBlockInput,
  MoveNlsInfoboxBlockMutation,
  MutationAddNlsInfoboxBlockArgs,
  MutationMoveNlsInfoboxBlockArgs,
  MutationRemoveNlsInfoboxBlockArgs,
  RemoveNlsInfoboxBlockInput,
  RemoveNlsInfoboxBlockMutation
} from "@reearth/services/gql";
import {
  ADD_NLSINFOBOX_BLOCK,
  MOVE_NLSINFOBOX_BLOCK,
  REMOVE_NLSINFOBOX_BLOCK
} from "@reearth/services/gql/queries/infobox";
import {
  applyAddNlsInfoboxBlockPayload,
  applyMoveNlsInfoboxBlockPayload,
  applyRemoveNlsInfoboxBlockPayload,
  useCollab
} from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

import { MutationReturn } from "../types";

/** Pass `sceneId` when the editor is bound to a scene so infobox block ops use collab WS. */
export const useInfoboxBlockMutations = (sceneIdForCollab?: string) => {
  const [, setNotification] = useNotification();
  const t = useT();
  const collab = useCollab();

  const [createInfoboxBlockMutation] = useMutation<
    AddNlsInfoboxBlockMutation,
    MutationAddNlsInfoboxBlockArgs
  >(ADD_NLSINFOBOX_BLOCK, { refetchQueries: ["GetScene"] });

  const createInfoboxBlock = useCallback(
    async (
      input: AddNlsInfoboxBlockInput
    ): Promise<MutationReturn<AddNlsInfoboxBlockMutation>> => {
      if (collab?.status === "open" && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyAddNlsInfoboxBlockPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            pluginId: input.pluginId,
            extensionId: input.extensionId,
            index: input.index ?? undefined,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully created a block!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await createInfoboxBlockMutation({
        variables: { input }
      });
      if (error || !data?.addNLSInfoboxBlock) {
        setNotification({ type: "error", text: t("Failed to create block.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully created a block!")
      });

      return { data, status: "success" };
    },
    [collab, createInfoboxBlockMutation, sceneIdForCollab, setNotification, t]
  );

  const [removeInfoboxBlockMutation] = useMutation<
    RemoveNlsInfoboxBlockMutation,
    MutationRemoveNlsInfoboxBlockArgs
  >(REMOVE_NLSINFOBOX_BLOCK, { refetchQueries: ["GetScene"] });

  const deleteInfoboxBlock = useCallback(
    async (
      input: RemoveNlsInfoboxBlockInput
    ): Promise<MutationReturn<RemoveNlsInfoboxBlockMutation>> => {
      if (collab?.status === "open" && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyRemoveNlsInfoboxBlockPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            infoboxBlockId: input.infoboxBlockId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "info",
            text: t("Block was successfully deleted.")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await removeInfoboxBlockMutation({
        variables: { input }
      });
      if (error || !data?.removeNLSInfoboxBlock) {
        setNotification({ type: "error", text: t("Failed to delete block.") });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Block was successfully deleted.")
      });

      return { data, status: "success" };
    },
    [collab, removeInfoboxBlockMutation, sceneIdForCollab, setNotification, t]
  );

  const [moveInfoboxBlockMutation] = useMutation<
    MoveNlsInfoboxBlockMutation,
    MutationMoveNlsInfoboxBlockArgs
  >(MOVE_NLSINFOBOX_BLOCK, { refetchQueries: ["GetScene"] });

  const moveInfoboxBlock = useCallback(
    async (
      input: MoveNlsInfoboxBlockInput
    ): Promise<MutationReturn<MoveNlsInfoboxBlockMutation>> => {
      if (collab?.status === "open" && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyMoveNlsInfoboxBlockPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            infoboxBlockId: input.infoboxBlockId,
            index: input.index,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "info",
            text: t("Block was successfully moved.")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await moveInfoboxBlockMutation({
        variables: { input }
      });
      if (error || !data?.moveNLSInfoboxBlock) {
        setNotification({ type: "error", text: t("Failed to move block.") });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Block was successfully moved.")
      });

      return { data, status: "success" };
    },
    [collab, moveInfoboxBlockMutation, sceneIdForCollab, setNotification, t]
  );

  return {
    createInfoboxBlock,
    deleteInfoboxBlock,
    moveInfoboxBlock
  };
};
