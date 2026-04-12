import { SwitchField } from "@reearth/app/ui/fields";
import PropertyItem from "@reearth/app/ui/fields/Properties";
import { NLSPhotoOverlay } from "@reearth/services/api/layer/types";
import { useT } from "@reearth/services/i18n/hooks";
import { styled } from "@reearth/services/theme";
import { FC } from "react";

import useHooks from "./hooks";

type Props = {
  selectedLayerId: string;
  photoOverlay?: NLSPhotoOverlay;
  sceneId?: string;
};

const PhotoOverlaySettings: FC<Props> = ({
  selectedLayerId,
  photoOverlay,
  sceneId
}) => {
  const t = useT();

  const {
    visibleItems,
    handlePhotoOverlayCreate,
    handlePhotoOverlayRemove
  } = useHooks({
    layerId: selectedLayerId,
    property: photoOverlay?.property,
    sceneId
  });

  return (
    <Wrapper>
      {visibleItems ? (
        <>
          {photoOverlay?.property?.id ? (
            <SwitchField
              title={t("Photo overlay enabled")}
              description={t("Disable photo overlay description")}
              value
              onChange={async (on) => {
                if (!on) await handlePhotoOverlayRemove();
              }}
            />
          ) : null}
          {visibleItems.map((i) =>
            photoOverlay?.property?.id ? (
              <PropertyItem
                key={i.id ?? ""}
                propertyId={photoOverlay.property.id}
                item={i}
                sceneId={sceneId}
              />
            ) : null
          )}
        </>
      ) : (
        <SwitchField
          title={t("Enable PhotoOverlay")}
          description={t("Show photo overlay when the user clicks on a layer")}
          value={false}
          onChange={handlePhotoOverlayCreate}
        />
      )}
    </Wrapper>
  );
};

const Wrapper = styled("div")(({ theme }) => ({
  display: "flex",
  flexDirection: "column",
  gap: theme.spacing.large
}));

export default PhotoOverlaySettings;
