import { useMutation } from "@apollo/client/react";
import { MutationReturn } from "@reearth/services/api/types";
import {
  applyAddNlsLayerSimplePayload,
  applyRemoveNlsLayerPayload,
  applyUpdateNlsLayerPayload,
  applyUpdateNlsLayersPayload,
  useCollab
} from "@reearth/services/collab";
import {
  AddNlsLayerSimpleMutation,
  MutationAddNlsLayerSimpleArgs,
  AddNlsLayerSimpleInput,
  UpdateNlsLayerInput,
  UpdateNlsLayerMutation,
  RemoveNlsLayerMutation,
  RemoveNlsLayerInput,
  UpdateNlsLayersInput,
  UpdateNlsLayersMutation
} from "@reearth/services/gql/__gen__/graphql";
import {
  ADD_NLSLAYERSIMPLE,
  UPDATE_NLSLAYER,
  REMOVE_NLSLAYER,
  UPDATE_NLSLAYERS
} from "@reearth/services/gql/queries/layer";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

export const useNLSLayerMutations = (sceneId?: string) => {
  const t = useT();
  const [, setNotification] = useNotification();
  const collab = useCollab();

  const [addNLSLayerSimpleMutation] = useMutation<
    AddNlsLayerSimpleMutation,
    MutationAddNlsLayerSimpleArgs
  >(ADD_NLSLAYERSIMPLE, {
    refetchQueries: ["GetScene"]
  });
  const addNLSLayerSimple = useCallback(
    async (
      input: AddNlsLayerSimpleInput
    ): Promise<MutationReturn<AddNlsLayerSimpleMutation>> => {
      if (collab?.status === "open") {
        const ok = collab.sendRaw(
          applyAddNlsLayerSimplePayload({
            sceneId: input.sceneId,
            title: input.title,
            layerType: input.layerType,
            config: input.config ?? undefined,
            index: input.index ?? undefined,
            visible: input.visible ?? undefined,
            schema: input.schema ?? undefined,
            dataSourceName: input.dataSourceName ?? undefined,
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
      const { data, error } = await addNLSLayerSimpleMutation({
        variables: { input }
      });
      if (error || !data?.addNLSLayerSimple?.layers?.id) {
        setNotification({ type: "error", text: t("Failed to add layer.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully added a new layer")
      });

      return { data, status: "success" };
    },
    [addNLSLayerSimpleMutation, collab, setNotification, t]
  );

  const [updateNLSLayerMutation] = useMutation(UPDATE_NLSLAYER, {
    refetchQueries: ["GetScene"]
  });
  const updateNLSLayer = useCallback(
    async (
      input: UpdateNlsLayerInput
    ): Promise<MutationReturn<UpdateNlsLayerMutation>> => {
      if (!input.layerId) return { status: "error" };
      if (collab?.status === "open" && sceneId) {
        const ok = collab.sendRaw(
          applyUpdateNlsLayerPayload({
            sceneId,
            layerId: input.layerId,
            index: input.index ?? undefined,
            name: input.name ?? undefined,
            visible: input.visible ?? undefined,
            config: input.config ?? undefined,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully updated the layer!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await updateNLSLayerMutation({
        variables: { input }
      });
      if (error || !data?.updateNLSLayer) {
        setNotification({
          type: "error",
          text: t("Failed to update the layer.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully updated the layer!")
      });

      return { data, status: "success" };
    },
    [collab, sceneId, updateNLSLayerMutation, t, setNotification]
  );

  const [updateNLSLayersMutation] = useMutation(UPDATE_NLSLAYERS, {
    refetchQueries: ["GetScene"]
  });
  const updateNLSLayers = useCallback(
    async (
      input: UpdateNlsLayersInput
    ): Promise<MutationReturn<UpdateNlsLayersMutation>> => {
      if (!input) return { status: "error" };
      if (collab?.status === "open" && sceneId) {
        const ok = collab.sendRaw(
          applyUpdateNlsLayersPayload({
            sceneId,
            layers: input.layers.map((l) => ({
              layerId: l.layerId,
              index: l.index ?? undefined,
              name: l.name ?? undefined,
              visible: l.visible ?? undefined,
              config: l.config ?? undefined
            })),
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully updated the layer!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await updateNLSLayersMutation({
        variables: { input }
      });
      if (error || !data?.updateNLSLayers) {
        setNotification({
          type: "error",
          text: t("Failed to update the layer.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully updated the layer!")
      });

      return { data, status: "success" };
    },
    [collab, sceneId, updateNLSLayersMutation, setNotification, t]
  );

  const [removeNLSLayerMutation] = useMutation(REMOVE_NLSLAYER, {
    refetchQueries: ["GetScene"]
  });
  const removeNLSLayer = useCallback(
    async (
      input: RemoveNlsLayerInput
    ): Promise<MutationReturn<RemoveNlsLayerMutation>> => {
      if (!input.layerId) return { status: "error" };
      if (collab?.status === "open" && sceneId) {
        const ok = collab.sendRaw(
          applyRemoveNlsLayerPayload({
            sceneId,
            layerId: input.layerId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully removed the layer!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await removeNLSLayerMutation({
        variables: { input }
      });
      if (error || !data?.removeNLSLayer) {
        setNotification({
          type: "error",
          text: t("Failed to remove the layer.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully removed the layer!")
      });

      return { data, status: "success" };
    },
    [collab, sceneId, removeNLSLayerMutation, t, setNotification]
  );

  return {
    addNLSLayerSimple,
    updateNLSLayer,
    updateNLSLayers,
    removeNLSLayer
  };
};
