import { useMutation } from "@apollo/client/react";
import { MutationReturn } from "@reearth/services/api/types";
import {
  applyChangeNlsCustomPropertyTitlePayload,
  applyRemoveNlsCustomPropertyPayload,
  applyUpdateNlsCustomPropertiesPayload,
  useCollab
} from "@reearth/services/collab";
import {
  UpdateCustomPropertySchemaInput,
  UpdateCustomPropertiesMutation,
  ChangeCustomPropertyTitleInput,
  ChangeCustomPropertyTitleMutation,
  RemoveCustomPropertyInput,
  RemoveCustomPropertyMutation
} from "@reearth/services/gql/__gen__/graphql";
import {
  UPDATE_CUSTOM_PROPERTY_SCHEMA,
  CHANGE_CUSTOM_PROPERTY_TITLE,
  REMOVE_CUSTOM_PROPERTY
} from "@reearth/services/gql/queries/layer";
import { useT } from "@reearth/services/i18n/hooks";
import { useNotification } from "@reearth/services/state";
import { useCallback } from "react";

/** Pass `sceneId` when the editor is bound to a scene so sketch custom property edits use collab WS. */
export const useNLSLayerCustomPropertyMutations = (
  sceneIdForCollab?: string
) => {
  const t = useT();
  const [, setNotification] = useNotification();
  const collab = useCollab();

  const [updateCustomPropertiesMutation] = useMutation(
    UPDATE_CUSTOM_PROPERTY_SCHEMA,
    {
      refetchQueries: ["GetScene"]
    }
  );
  const updateCustomProperties = useCallback(
    async (
      input: UpdateCustomPropertySchemaInput
    ): Promise<MutationReturn<UpdateCustomPropertiesMutation>> => {
      if (!input.layerId) return { status: "error" };
      if (collab && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyUpdateNlsCustomPropertiesPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            schema: input.schema ?? {},
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully updated the custom property schema!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await updateCustomPropertiesMutation({
        variables: { input }
      });

      if (error || !data?.updateCustomProperties) {
        setNotification({
          type: "error",
          text: t("Failed to update the custom property schema.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully updated the custom property schema!")
      });

      return { data, status: "success" };
    },
    [
      collab,
      sceneIdForCollab,
      updateCustomPropertiesMutation,
      setNotification,
      t
    ]
  );

  const [changeCustomPropertyTitleMutation] = useMutation(
    CHANGE_CUSTOM_PROPERTY_TITLE,
    {
      refetchQueries: ["GetScene"]
    }
  );

  const changeCustomPropertyTitle = useCallback(
    async (
      input: ChangeCustomPropertyTitleInput
    ): Promise<MutationReturn<ChangeCustomPropertyTitleMutation>> => {
      if (!input.layerId) return { status: "error" };
      if (collab && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyChangeNlsCustomPropertyTitlePayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            schema: input.schema ?? {},
            oldTitle: input.oldTitle,
            newTitle: input.newTitle,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully updated the custom property title!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await changeCustomPropertyTitleMutation({
        variables: { input }
      });
      if (error || !data?.changeCustomPropertyTitle) {
        setNotification({
          type: "error",
          text: t("Failed to update the custom property title.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully updated the custom property title!")
      });

      return { data, status: "success" };
    },
    [
      collab,
      sceneIdForCollab,
      changeCustomPropertyTitleMutation,
      setNotification,
      t
    ]
  );

  const [removeCustomPropertyMutation] = useMutation(REMOVE_CUSTOM_PROPERTY, {
    refetchQueries: ["GetScene"]
  });
  const removeCustomProperty = useCallback(
    async (
      input: RemoveCustomPropertyInput
    ): Promise<MutationReturn<RemoveCustomPropertyMutation>> => {
      if (!input.layerId) return { status: "error" };
      if (collab && sceneIdForCollab) {
        const ok = collab.sendRaw(
          applyRemoveNlsCustomPropertyPayload({
            sceneId: sceneIdForCollab,
            layerId: input.layerId,
            schema: input.schema ?? {},
            removedTitle: input.removedTitle,
            baseSceneRev: collab.remoteSceneRev
          })
        );
        if (ok) {
          setNotification({
            type: "success",
            text: t("Successfully removed the custom property!")
          });
          return { status: "success" as const };
        }
      }
      const { data, error } = await removeCustomPropertyMutation({
        variables: { input }
      });
      if (error || !data?.removeCustomProperty) {
        setNotification({
          type: "error",
          text: t("Failed to remove the custom property.")
        });

        return { status: "error", error };
      }
      setNotification({
        type: "success",
        text: t("Successfully removed the custom property!")
      });

      return { data, status: "success" };
    },
    [collab, sceneIdForCollab, removeCustomPropertyMutation, setNotification, t]
  );

  return {
    updateCustomProperties,
    changeCustomPropertyTitle,
    removeCustomProperty
  };
};
