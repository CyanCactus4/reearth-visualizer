import { useMutation } from "@apollo/client/react";
import { MutationReturn } from "@reearth/services/api/types";
import {
  applyAddNlsGeojsonFeaturePayload,
  applyDeleteNlsGeojsonFeaturePayload,
  applyUpdateNlsGeojsonFeaturePayload,
  useCollab
} from "@reearth/services/collab";
import {
  AddGeoJsonFeatureInput,
  AddGeoJsonFeatureMutation,
  DeleteGeoJsonFeatureInput,
  DeleteGeoJsonFeatureMutation,
  MutationAddGeoJsonFeatureArgs,
  MutationDeleteGeoJsonFeatureArgs,
  UpdateGeoJsonFeatureInput,
  UpdateGeoJsonFeatureMutation
} from "@reearth/services/gql/__gen__/graphql";
import {
  ADD_GEOJSON_FEATURE,
  DELETE_GEOJSON_FEATURE,
  UPDATE_GEOJSON_FEATURE
} from "@reearth/services/gql/queries/featureCollection";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

/** Pass `sceneId` when the editor is bound to a scene so sketch GeoJSON edits use collab WS. */
export const useFeatureCollectionMutations = (
  sceneIdForCollab?: string
) => {
  const t = useT();
  const [, setNotification] = useNotification();
  const collab = useCollab();

  const [addGeoJsonFeatureMutation] = useMutation<
    AddGeoJsonFeatureMutation,
    MutationAddGeoJsonFeatureArgs
  >(ADD_GEOJSON_FEATURE, {
    refetchQueries: ["GetScene"]
  });

  const addGeoJsonFeature = useCallback(
    async (
      input: AddGeoJsonFeatureInput
    ): Promise<MutationReturn<AddGeoJsonFeatureMutation>> => {
      if (collab?.status === "open" && sceneIdForCollab && input.geometry) {
        const ok = collab.sendRaw(
          applyAddNlsGeojsonFeaturePayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            type: input.type,
            geometry: input.geometry,
            properties: input.properties ?? undefined,
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
      const { data, error } = await addGeoJsonFeatureMutation({
        variables: { input }
      });
      if (error || !data?.addGeoJSONFeature.id) {
        setNotification({ type: "error", text: t("Failed to add layer.") });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully added a new layer")
      });
      return { data, status: "success" };
    },
    [addGeoJsonFeatureMutation, collab, sceneIdForCollab, setNotification, t]
  );

  const [updateGeoJsonFeatureMutation] = useMutation(UPDATE_GEOJSON_FEATURE, {
    refetchQueries: ["GetScene"]
  });

  const updateGeoJSONFeature = useCallback(
    async (
      input: UpdateGeoJsonFeatureInput
    ): Promise<MutationReturn<UpdateGeoJsonFeatureMutation>> => {
      if (!input.layerId) return { status: "error" };
      const hasGeom = input.geometry !== undefined && input.geometry !== null;
      const hasProps =
        input.properties !== undefined && input.properties !== null;
      if (collab?.status === "open" && sceneIdForCollab && (hasGeom || hasProps)) {
        const ok = collab.sendRaw(
          applyUpdateNlsGeojsonFeaturePayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            featureId: input.featureId,
            ...(hasGeom ? { geometry: input.geometry } : {}),
            ...(hasProps ? { properties: input.properties } : {}),
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
      const { data, error } = await updateGeoJsonFeatureMutation({
        variables: { input }
      });
      if (error || !data?.updateGeoJSONFeature) {
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
    [collab, sceneIdForCollab, updateGeoJsonFeatureMutation, setNotification, t]
  );

  const [deleteGeoJsonFeatureMutation] = useMutation<
    DeleteGeoJsonFeatureMutation,
    MutationDeleteGeoJsonFeatureArgs
  >(DELETE_GEOJSON_FEATURE, {
    refetchQueries: ["GetScene"]
  });

  const deleteGeoJSONFeature = useCallback(
    async (
      input: DeleteGeoJsonFeatureInput
    ): Promise<MutationReturn<DeleteGeoJsonFeatureMutation>> => {
      if (!input.layerId || !input.featureId) return { status: "error" };
      if (collab?.status === "open" && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyDeleteNlsGeojsonFeaturePayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            featureId: input.featureId,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully deleted the feature!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await deleteGeoJsonFeatureMutation({
        variables: { input }
      });
      if (error || !data?.deleteGeoJSONFeature) {
        setNotification({
          type: "error",
          text: t("Failed to delete the feature.")
        });
        return { status: "error", error };
      }

      setNotification({
        type: "success",
        text: t("Successfully deleted the feature!")
      });
      return { data, status: "success" };
    },
    [collab, deleteGeoJsonFeatureMutation, sceneIdForCollab, t, setNotification]
  );

  return {
    addGeoJsonFeature,
    updateGeoJSONFeature,
    deleteGeoJSONFeature
  };
};
