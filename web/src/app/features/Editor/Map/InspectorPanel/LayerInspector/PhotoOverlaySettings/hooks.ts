import { filterVisibleItems } from "@reearth/app/ui/fields/utils";
import { usePhotoOverlayMutations } from "@reearth/services/api/photoOverlay";
import type { Item } from "@reearth/services/api/property";
import { convert } from "@reearth/services/api/property/utils";
import type { PropertyFragmentFragment } from "@reearth/services/gql";
import { useCallback, useMemo } from "react";

export default ({
  layerId,
  property,
  sceneId
}: {
  layerId: string;
  property?: PropertyFragmentFragment | null;
  sceneId?: string;
}) => {
  const { createNLSPhotoOverlay, removeNLSPhotoOverlay } =
    usePhotoOverlayMutations(sceneId);

  const visibleItems: Item[] | undefined = useMemo(
    () => filterVisibleItems(convert(property)),
    [property]
  );

  const handlePhotoOverlayCreate = useCallback(
    async (enabled?: boolean) => {
      if (enabled !== true) return;
      if (!property) {
        await createNLSPhotoOverlay({ layerId });
      }
    },
    [layerId, property, createNLSPhotoOverlay]
  );

  const handlePhotoOverlayRemove = useCallback(async () => {
    await removeNLSPhotoOverlay({ layerId });
  }, [layerId, removeNLSPhotoOverlay]);

  return {
    visibleItems,
    handlePhotoOverlayCreate,
    handlePhotoOverlayRemove
  };
};
