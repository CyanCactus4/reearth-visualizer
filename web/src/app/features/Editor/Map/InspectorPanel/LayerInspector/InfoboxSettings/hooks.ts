import { filterVisibleItems } from "@reearth/app/ui/fields/utils";
import { useInfoboxMutations } from "@reearth/services/api/infobox";
import type { Item } from "@reearth/services/api/property";
import { convert } from "@reearth/services/api/property/utils";
import { PropertyFragmentFragment } from "@reearth/services/gql";
import { useCallback, useMemo } from "react";

export default ({
  layerId,
  property,
  sceneId
}: {
  layerId: string;
  property?: PropertyFragmentFragment | null | undefined;
  sceneId?: string;
}) => {
  const { createNLSInfobox, removeNLSInfobox } = useInfoboxMutations(sceneId);

  const visibleItems: Item[] | undefined = useMemo(
    () => filterVisibleItems(convert(property)),
    [property]
  );

  const handleInfoboxCreate = useCallback(async () => {
    if (!property) {
      await createNLSInfobox({ layerId });
    }
  }, [layerId, property, createNLSInfobox]);

  const handleInfoboxRemove = useCallback(async () => {
    await removeNLSInfobox({ layerId });
  }, [layerId, removeNLSInfobox]);

  return {
    visibleItems,
    handleInfoboxCreate,
    handleInfoboxRemove
  };
};
