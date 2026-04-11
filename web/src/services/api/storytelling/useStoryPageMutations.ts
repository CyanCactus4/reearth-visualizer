import { useMutation } from "@apollo/client/react";
import { MutationReturn } from "@reearth/services/api/types";
import {
  applyCreateStoryPagePayload,
  applyMoveStoryPagePayload,
  applyRemoveStoryPagePayload,
  useCollab
} from "@reearth/services/collab";
import {
  CreateStoryPageInput,
  CreateStoryPageMutation,
  DeleteStoryPageInput,
  DeleteStoryPageMutation,
  MoveStoryPageInput,
  MoveStoryPageMutation,
  MutationCreateStoryPageArgs,
  MutationMoveStoryPageArgs,
  MutationRemoveStoryPageArgs,
  MutationUpdateStoryPageArgs,
  UpdateStoryPageInput,
  UpdateStoryPageMutation
} from "@reearth/services/gql/__gen__/graphql";
import {
  CREATE_STORY_PAGE,
  DELETE_STORY_PAGE,
  MOVE_STORY_PAGE,
  UPDATE_STORY_PAGE
} from "@reearth/services/gql/queries/storytelling";
import { useT } from "@reearth/services/i18n/hooks";
import { useCallback } from "react";

import { useNotification } from "../../state";

/** Pass `sceneId` when the editor is bound to a scene so `moveStoryPage` can use collab WS (input has no sceneId). */
export const useStoryPageMutations = (sceneIdForCollab?: string) => {
  const [, setNotification] = useNotification();
  const t = useT();
  const collab = useCollab();

  const [createStoryPageMutation] = useMutation<
    CreateStoryPageMutation,
    MutationCreateStoryPageArgs
  >(CREATE_STORY_PAGE, { refetchQueries: ["GetScene"] });

  const createStoryPage = useCallback(
    async (
      input: CreateStoryPageInput
    ): Promise<MutationReturn<CreateStoryPageMutation>> => {
      if (collab?.status === "open" && input.sceneId) {
        const ok = collab.sendRaw(
          applyCreateStoryPagePayload({
            sceneId: input.sceneId,
            storyId: input.storyId,
            title: input.title ?? undefined,
            swipeable: input.swipeable ?? undefined,
            layers: input.layers?.length ? [...input.layers] : undefined,
            swipeableLayers: input.swipeableLayers?.length
              ? [...input.swipeableLayers]
              : undefined,
            index: input.index ?? undefined,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully created a page!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await createStoryPageMutation({
        variables: {
          input
        }
      });
      if (error || !data?.createStoryPage?.story?.id) {
        setNotification({ type: "error", text: t("Failed to create page.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully created a page!")
      });

      return { data, status: "success" };
    },
    [collab, createStoryPageMutation, setNotification, t]
  );

  const [deleteStoryPageMutation] = useMutation<
    DeleteStoryPageMutation,
    MutationRemoveStoryPageArgs
  >(DELETE_STORY_PAGE, { refetchQueries: ["GetScene"] });

  const deleteStoryPage = useCallback(
    async (
      input: DeleteStoryPageInput
    ): Promise<MutationReturn<DeleteStoryPageMutation>> => {
      if (collab?.status === "open" && input.sceneId) {
        const ok = collab.sendRaw(
          applyRemoveStoryPagePayload({
            sceneId: input.sceneId,
            storyId: input.storyId,
            pageId: input.pageId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "info",
            text: t("Page was successfully deleted.")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await deleteStoryPageMutation({
        variables: {
          input
        }
      });
      if (error || !data?.removeStoryPage?.story?.id) {
        setNotification({ type: "error", text: t("Failed to delete page.") });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Page was successfully deleted.")
      });

      return { data, status: "success" };
    },
    [collab, deleteStoryPageMutation, setNotification, t]
  );

  const [moveStoryPageMutation] = useMutation<
    MoveStoryPageMutation,
    MutationMoveStoryPageArgs
  >(MOVE_STORY_PAGE, { refetchQueries: ["GetScene"] });

  const moveStoryPage = useCallback(
    async (
      input: MoveStoryPageInput
    ): Promise<MutationReturn<MoveStoryPageMutation>> => {
      if (collab?.status === "open" && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyMoveStoryPagePayload({
            sceneId: sceneIdForCollab,
            storyId: input.storyId,
            pageId: input.pageId,
            index: input.index,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "info",
            text: t("Page was successfully moved.")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await moveStoryPageMutation({
        variables: {
          input
        }
      });
      if (error || !data?.moveStoryPage?.story?.id) {
        setNotification({ type: "error", text: t("Failed to move page.") });

        return { status: "error", error };
      }
      setNotification({
        type: "info",
        text: t("Page was successfully moved.")
      });

      return { data, status: "success" };
    },
    [collab, moveStoryPageMutation, sceneIdForCollab, setNotification, t]
  );

  const [updateStoryPageMutation] = useMutation<
    UpdateStoryPageMutation,
    MutationUpdateStoryPageArgs
  >(UPDATE_STORY_PAGE, { refetchQueries: ["GetScene"] });

  const updateStoryPage = useCallback(
    async (
      input: UpdateStoryPageInput
    ): Promise<MutationReturn<UpdateStoryPageMutation>> => {
      const { data, error } = await updateStoryPageMutation({
        variables: {
          input
        }
      });
      if (error || !data?.updateStoryPage?.story?.id) {
        setNotification({ type: "error", text: t("Failed to update page.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully updated a page!")
      });

      return { data, status: "success" };
    },
    [updateStoryPageMutation, setNotification, t]
  );
  return {
    createStoryPage,
    deleteStoryPage,
    moveStoryPage,
    updateStoryPage
  };
};
