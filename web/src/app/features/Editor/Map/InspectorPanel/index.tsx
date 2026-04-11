import { Panel, PanelProps } from "@reearth/app/ui/layout";
import { CollabLockGate } from "@reearth/services/collab";
import { useT } from "@reearth/services/i18n/hooks";
import { FC, useMemo } from "react";

import { useMapPage } from "../context";

import LayerInspector from "./LayerInspector";
import SceneSettings from "./SceneSettings";

type Props = Pick<PanelProps, "showCollapseArea" | "areaRef">;

const InspectorPanel: FC<Props> = ({ areaRef, showCollapseArea }) => {
  const {
    scene,
    layers,
    layerStyles,
    sceneId,
    selectedSceneSetting,
    sceneSettings,
    selectedLayer,
    selectedFeature,
    selectedSketchFeature,
    handleFlyTo,
    handleLayerConfigUpdate,
    handleLayerNameUpdate,
    handleGeoJsonFeatureUpdate
  } = useMapPage();

  const t = useT();

  const scenePropertyId = useMemo(
    () => scene?.property?.id,
    [scene?.property?.id]
  );

  return (
    <Panel
      title={t("Inspector")}
      dataTestid="editor-map-inspector-panel"
      storageId="editor-map-inspector-panel"
      extend
      alwaysOpen
      noPadding={!!selectedLayer}
      areaRef={areaRef}
      showCollapseArea={showCollapseArea}
    >
      {!!selectedSceneSetting && scenePropertyId && sceneId && (
        <CollabLockGate resource="scene" id={sceneId}>
          <SceneSettings
            propertyId={scenePropertyId}
            propertyItems={sceneSettings}
            onFlyTo={handleFlyTo}
          />
        </CollabLockGate>
      )}
      {selectedLayer && (
        <LayerInspector
          layerStyles={layerStyles}
          layers={layers}
          sceneId={sceneId}
          selectedLayer={selectedLayer}
          selectedFeature={selectedFeature}
          selectedSketchFeature={selectedSketchFeature}
          onLayerConfigUpdate={handleLayerConfigUpdate}
          onGeoJsonFeatureUpdate={handleGeoJsonFeatureUpdate}
          onLayerNameUpdate={handleLayerNameUpdate}
        />
      )}
    </Panel>
  );
};

export default InspectorPanel;
