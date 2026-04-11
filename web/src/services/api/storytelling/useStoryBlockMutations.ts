import { useMutation } from "@apollo/client/react";
import { MutationReturn } from "@reearth/services/api/types";
import {
  applyCreateStoryBlockPayload,
  applyMoveStoryBlockPayload,
  applyRemoveStoryBlockPayload,
  useCollab
} from "@reearth/services/collab";
import {
  CreateStoryBlockInput,
  CreateStoryBlockMutation,
  MoveStoryBlockInput,
  MoveStoryBlockMutation,
  MutationCreateStoryBlockArgs,
  MutationMoveStoryBlockArgs,
  MutationRemoveStoryBlockArgs,
  RemoveStoryBlockInput,
  RemoveStoryBlockMutation
} from "@reearth/services/gql/__gen__/graphql";
import {
  CREATE_STORY_BLOCK,
  MOVE_STORY_BLOCK,
  REMOVE_STORY_BLOCK
} from "@reearth/services/gql/queries/storytelling";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

export const useStoryBlockMutations = (sceneId?: string) => {
  const [, setNotification] = useNotification();
  const t = useT();
  const collab = useCollab();

  const [createStoryBlockMutation] = useMutation<
    CreateStoryBlockMutation,
    MutationCreateStoryBlockArgs
  >(CREATE_STORY_BLOCK, { refetchQueries: ["GetScene"] });

  const createStoryBlock = useCallback(
    async (
      input: CreateStoryBlockInput
    ): Promise<MutationReturn<CreateStoryBlockMutation>> => {
      if (collab?.status === "open" && sceneId) {
        const ok = collab.sendRaw(
          applyCreateStoryBlockPayload({
            sceneId,
            storyId: input.storyId,
            pageId: input.pageId,
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
      const { data, error } = await createStoryBlockMutation({
        variables: { input }
      });
      if (error || !data?.createStoryBlock) {
        setNotification({ type: "error", text: t("Failed to create block.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully created a block!")
      });

      return { data, status: "success" };
    },
    [collab, createStoryBlockMutation, sceneId, setNotification, t]
  );

  const [removeStoryBlockMutation] = useMutation<
    RemoveStoryBlockMutation,
    MutationRemoveStoryBlockArgs
  >(REMOVE_STORY_BLOCK, { refetchQueries: ["GetScene"] });

  const deleteStoryBlock = useCallback(
    async (
      input: RemoveStoryBlockInput
    ): Promise<MutationReturn<RemoveStoryBlockMutation>> => {
      if (collab?.status === "open" && sceneId) {
        const ok = collab.sendRaw(
          applyRemoveStoryBlockPayload({
            sceneId,
            storyId: input.storyId,
            pageId: input.pageId,
            blockId: input.blockId,
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
      const { data, error } = await removeStoryBlockMutation({
        variables: { input }
      });
      if (error || !data?.removeStoryBlock) {
        setNotification({ type: "error", text: t("Failed to delete block.") });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Block was successfully deleted.")
      });

      return { data, status: "success" };
    },
    [collab, removeStoryBlockMutation, sceneId, setNotification, t]
  );

  const [moveStoryBlockMutation] = useMutation<
    MoveStoryBlockMutation,
    MutationMoveStoryBlockArgs
  >(MOVE_STORY_BLOCK, { refetchQueries: ["GetScene"] });

  const moveStoryBlock = useCallback(
    async (
      input: MoveStoryBlockInput
    ): Promise<MutationReturn<MoveStoryBlockMutation>> => {
      if (collab?.status === "open" && sceneId) {
        const ok = collab.sendRaw(
          applyMoveStoryBlockPayload({
            sceneId,
            storyId: input.storyId,
            pageId: input.pageId,
            blockId: input.blockId,
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
      const { data, error } = await moveStoryBlockMutation({
        variables: { input }
      });
      if (error || !data?.moveStoryBlock) {
        setNotification({ type: "error", text: t("Failed to move block.") });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Block was successfully moved.")
      });

      return { data, status: "success" };
    },
    [collab, moveStoryBlockMutation, sceneId, setNotification, t]
  );
  return {
    createStoryBlock,
    deleteStoryBlock,
    moveStoryBlock
  };
};
