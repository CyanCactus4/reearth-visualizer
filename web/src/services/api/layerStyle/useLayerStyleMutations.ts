import { useMutation } from "@apollo/client/react";
import { MutationReturn } from "@reearth/services/api/types";
import {
  applyAddStylePayload,
  applyRemoveStylePayload,
  applyUpdateStylePayload,
  useCollab
} from "@reearth/services/collab";
import {
  AddStyleMutation,
  MutationAddStyleArgs,
  AddStyleInput,
  UpdateStyleInput,
  UpdateStyleMutation,
  RemoveStyleMutation,
  RemoveStyleInput
} from "@reearth/services/gql/__gen__/graphql";
import {
  ADD_LAYERSTYLE,
  REMOVE_LAYERSTYLE,
  UPDATE_LAYERSTYLE
} from "@reearth/services/gql/queries/layerStyle";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

export const useLayerStyleMutations = (sceneId?: string) => {
  const t = useT();
  const [, setNotification] = useNotification();
  const collab = useCollab();

  const [addLayerStyleMutation] = useMutation<
    AddStyleMutation,
    MutationAddStyleArgs
  >(ADD_LAYERSTYLE, {
    refetchQueries: ["GetScene"]
  });
  const addLayerStyle = useCallback(
    async (input: AddStyleInput): Promise<MutationReturn<AddStyleMutation>> => {
      const sid = input.sceneId ?? sceneId;
      if (collab && sid) {
        const ok = collab.sendRaw(
          applyAddStylePayload({
            sceneId: sid,
            name: input.name,
            value: input.value,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully added a new layer style!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await addLayerStyleMutation({
        variables: { input }
      });
      if (error || !data?.addStyle?.style?.id) {
        setNotification({
          type: "error",
          text: t("Failed to add layer style.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully added a new layer style!")
      });

      return { data, status: "success" };
    },
    [addLayerStyleMutation, collab, sceneId, setNotification, t]
  );

  const [updateLayerStyleMutation] = useMutation(UPDATE_LAYERSTYLE, {
    refetchQueries: ["GetScene"]
  });
  const updateLayerStyle = useCallback(
    async (
      input: UpdateStyleInput
    ): Promise<MutationReturn<UpdateStyleMutation>> => {
      if (!input.styleId) return { status: "error" };
      if (collab && sceneId) {
        const ok = collab.sendRaw(
          applyUpdateStylePayload({
            sceneId,
            styleId: input.styleId,
            name: input.name ?? undefined,
            value: input.value ?? undefined,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully updated a the layerStyle!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await updateLayerStyleMutation({
        variables: { input }
      });
      if (error || !data?.updateStyle) {
        setNotification({
          type: "error",
          text: t("Failed to update the layerStyle.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully updated a the layerStyle!")
      });

      return { data, status: "success" };
    },
    [collab, sceneId, updateLayerStyleMutation, setNotification, t]
  );

  const [removeLayerStyleMutation] = useMutation(REMOVE_LAYERSTYLE, {
    refetchQueries: ["GetScene"]
  });
  const removeLayerStyle = useCallback(
    async (
      input: RemoveStyleInput
    ): Promise<MutationReturn<RemoveStyleMutation>> => {
      if (!input.styleId) return { status: "error" };
      if (collab && sceneId) {
        const ok = collab.sendRaw(
          applyRemoveStylePayload({
            sceneId,
            styleId: input.styleId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully deleted the layer style!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await removeLayerStyleMutation({
        variables: { input }
      });
      if (error || !data?.removeStyle) {
        setNotification({
          type: "error",
          text: t("Failed to delete the layer style.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully deleted the layer style!")
      });

      return { data, status: "success" };
    },
    [collab, sceneId, removeLayerStyleMutation, setNotification, t]
  );

  return {
    addLayerStyle,
    updateLayerStyle,
    removeLayerStyle
  };
};
