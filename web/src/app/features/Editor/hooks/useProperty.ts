import { usePropertyMutations } from "@reearth/services/api/property";
import { useCallback } from "react";

type Props = {
  sceneId: string;
};

export default ({ sceneId }: Props) => {
  const { updatePropertyValue } = usePropertyMutations(sceneId);
  const handlePropertyValueUpdate = useCallback(
    async (
      propertyId?: string,
      schemaItemId?: string,
      fieldId?: string,
      itemId?: string,
      vt?: unknown,
      v?: unknown
    ) => {
      if (!propertyId || !schemaItemId || !fieldId || !vt) return;
      await updatePropertyValue(
        propertyId,
        schemaItemId,
        itemId,
        fieldId,
        "en",
        v as string | number | boolean | unknown[] | undefined,
        vt as keyof import("@reearth/core").ValueTypes
      );
    },
    [updatePropertyValue]
  );

  return {
    handlePropertyValueUpdate
  };
};
