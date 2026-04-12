import { WidgetAlignSystemType } from "@reearth/services/gql";
import { describe, expect, it } from "vitest";

import {
  alignSystemForCollab,
  applyAddNlsInfoboxBlockPayload,
  applyAddNlsLayerSimplePayload,
  applyAddStylePayload,
  applyAddWidgetPayload,
  applyCreateNlsInfoboxPayload,
  applyAddNlsGeojsonFeaturePayload,
  applyChangeNlsCustomPropertyTitlePayload,
  applyCreateNlsPhotoOverlayPayload,
  applyDeleteNlsGeojsonFeaturePayload,
  applyRemoveNlsCustomPropertyPayload,
  applyRemoveNlsInfoboxPayload,
  applyRemoveNlsPhotoOverlayPayload,
  applyUpdateNlsCustomPropertiesPayload,
  applyUpdateNlsGeojsonFeaturePayload,
  applyCreateStoryBlockPayload,
  applyCreateStoryPagePayload,
  applyDuplicateStoryPagePayload,
  applyMoveNlsInfoboxBlockPayload,
  applyMoveStoryBlockPayload,
  applyMoveStoryPagePayload,
  applyRemoveStoryBlockPayload,
  applyRemoveStoryPagePayload,
  applyRemoveNlsInfoboxBlockPayload,
  applyRemoveNlsLayerPayload,
  applyRemoveStylePayload,
  applyUpdateNlsLayerPayload,
  applyUpdateNlsLayersPayload,
  applyUpdatePropertyValuePayload,
  applyUpdateStylePayload,
  applyUpdateStoryPagePayload,
  applyRemoveWidgetPayload,
  applyUpdateWidgetPayload
} from "./applyMessages";

describe("alignSystemForCollab", () => {
  it("maps GraphQL enum to server strings", () => {
    expect(alignSystemForCollab(WidgetAlignSystemType.Desktop)).toBe(
      "desktop"
    );
    expect(alignSystemForCollab(WidgetAlignSystemType.Mobile)).toBe("mobile");
  });
});

describe("apply payloads", () => {
  it("builds update_widget apply envelope", () => {
    const s = applyUpdateWidgetPayload({
      sceneId: "sc1",
      widgetId: "w1",
      type: WidgetAlignSystemType.Desktop,
      enabled: true,
      location: { zone: "outer", section: "left", area: "top" }
    });
    expect(JSON.parse(s)).toEqual({
      v: 1,
      t: "apply",
      d: {
        kind: "update_widget",
        sceneId: "sc1",
        alignSystem: "desktop",
        widgetId: "w1",
        enabled: true,
        location: { zone: "outer", section: "left", area: "top" }
      }
    });
  });

  it("builds remove_widget apply envelope", () => {
    const s = applyRemoveWidgetPayload({
      sceneId: "sc1",
      widgetId: "w1",
      type: WidgetAlignSystemType.Mobile
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "remove_widget",
      sceneId: "sc1",
      alignSystem: "mobile",
      widgetId: "w1"
    });
  });

  it("builds add_widget apply envelope", () => {
    const s = applyAddWidgetPayload({
      sceneId: "sc1",
      type: WidgetAlignSystemType.Desktop,
      pluginId: "plug~1.0.0",
      extensionId: "ext1"
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "add_widget",
      sceneId: "sc1",
      alignSystem: "desktop",
      pluginId: "plug~1.0.0",
      extensionId: "ext1"
    });
  });

  it("includes baseSceneRev when provided", () => {
    const s = applyUpdateWidgetPayload({
      sceneId: "sc1",
      widgetId: "w1",
      type: WidgetAlignSystemType.Desktop,
      baseSceneRev: 42
    });
    expect(JSON.parse(s).d.baseSceneRev).toBe(42);
  });

  it("includes entityClocks when provided", () => {
    const s = applyUpdateWidgetPayload({
      sceneId: "sc1",
      widgetId: "w1",
      type: WidgetAlignSystemType.Desktop,
      entityClocks: { enabled: 2, layout: 1 },
      extended: true
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "update_widget",
      entityClocks: { enabled: 2, layout: 1 },
      extended: true
    });
  });

  it("builds move_story_block apply envelope", () => {
    const s = applyMoveStoryBlockPayload({
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      blockId: "bk1",
      index: 2,
      baseSceneRev: 9
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "move_story_block",
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      blockId: "bk1",
      index: 2,
      baseSceneRev: 9
    });
  });

  it("builds create_story_block apply envelope", () => {
    const s = applyCreateStoryBlockPayload({
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      pluginId: "p~1",
      extensionId: "e1",
      index: 0,
      baseSceneRev: 3
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "create_story_block",
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      pluginId: "p~1",
      extensionId: "e1",
      index: 0,
      baseSceneRev: 3
    });
  });

  it("builds remove_story_block apply envelope", () => {
    const s = applyRemoveStoryBlockPayload({
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      blockId: "bk9"
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "remove_story_block",
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      blockId: "bk9"
    });
  });

  it("builds create_story_page apply envelope", () => {
    const s = applyCreateStoryPagePayload({
      sceneId: "sc1",
      storyId: "st1",
      title: "Intro",
      swipeable: true,
      layers: ["ly1"],
      index: 1,
      baseSceneRev: 5
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "create_story_page",
      sceneId: "sc1",
      storyId: "st1",
      title: "Intro",
      swipeable: true,
      layers: ["ly1"],
      index: 1,
      baseSceneRev: 5
    });
  });

  it("builds remove_story_page apply envelope", () => {
    const s = applyRemoveStoryPagePayload({
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg9"
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "remove_story_page",
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg9"
    });
  });

  it("builds move_story_page apply envelope", () => {
    const s = applyMoveStoryPagePayload({
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg9",
      index: 0
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "move_story_page",
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg9",
      index: 0
    });
  });

  it("builds duplicate_story_page apply envelope", () => {
    const s = applyDuplicateStoryPagePayload({
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      baseSceneRev: 7
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "duplicate_story_page",
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      baseSceneRev: 7
    });
  });

  it("builds update_story_page apply envelope", () => {
    const s = applyUpdateStoryPagePayload({
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      title: "T",
      swipeable: false,
      layers: [],
      index: 2,
      baseSceneRev: 11
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "update_story_page",
      sceneId: "sc1",
      storyId: "st1",
      pageId: "pg1",
      title: "T",
      swipeable: false,
      layers: [],
      index: 2,
      baseSceneRev: 11
    });
  });

  it("builds NLS layer apply envelopes", () => {
    const add = applyAddNlsLayerSimplePayload({
      sceneId: "sc1",
      title: "L1",
      layerType: "simple",
      config: { a: 1 },
      index: 0,
      baseSceneRev: 3
    });
    expect(JSON.parse(add).d).toMatchObject({
      kind: "add_nls_layer_simple",
      sceneId: "sc1",
      title: "L1",
      layerType: "simple",
      config: { a: 1 },
      index: 0,
      baseSceneRev: 3
    });

    const rm = applyRemoveNlsLayerPayload({
      sceneId: "sc1",
      layerId: "ly1",
      baseSceneRev: 4
    });
    expect(JSON.parse(rm).d).toMatchObject({
      kind: "remove_nls_layer",
      sceneId: "sc1",
      layerId: "ly1",
      baseSceneRev: 4
    });

    const upd = applyUpdateNlsLayerPayload({
      sceneId: "sc1",
      layerId: "ly1",
      name: "N",
      visible: false,
      baseSceneRev: 5
    });
    expect(JSON.parse(upd).d).toMatchObject({
      kind: "update_nls_layer",
      sceneId: "sc1",
      layerId: "ly1",
      name: "N",
      visible: false,
      baseSceneRev: 5
    });

    const batch = applyUpdateNlsLayersPayload({
      sceneId: "sc1",
      layers: [
        { layerId: "a", index: 0 },
        { layerId: "b", index: 1 }
      ],
      baseSceneRev: 6
    });
    expect(JSON.parse(batch).d).toMatchObject({
      kind: "update_nls_layers",
      sceneId: "sc1",
      layers: [{ layerId: "a", index: 0 }, { layerId: "b", index: 1 }],
      baseSceneRev: 6
    });
  });

  it("builds layer style (scene style) apply envelopes", () => {
    const add = applyAddStylePayload({
      sceneId: "sc1",
      name: "S1",
      value: { fillColor: "#fff" },
      baseSceneRev: 1
    });
    expect(JSON.parse(add).d).toMatchObject({
      kind: "add_style",
      sceneId: "sc1",
      name: "S1",
      value: { fillColor: "#fff" },
      baseSceneRev: 1
    });

    const upd = applyUpdateStylePayload({
      sceneId: "sc1",
      styleId: "st1",
      name: "Renamed",
      baseSceneRev: 2
    });
    expect(JSON.parse(upd).d).toMatchObject({
      kind: "update_style",
      sceneId: "sc1",
      styleId: "st1",
      name: "Renamed",
      baseSceneRev: 2
    });

    const rm = applyRemoveStylePayload({
      sceneId: "sc1",
      styleId: "st1",
      baseSceneRev: 3
    });
    expect(JSON.parse(rm).d).toMatchObject({
      kind: "remove_style",
      sceneId: "sc1",
      styleId: "st1",
      baseSceneRev: 3
    });
  });

  it("builds update_property_value apply envelope", () => {
    const s = applyUpdatePropertyValuePayload({
      sceneId: "sc1",
      propertyId: "pr1",
      schemaGroupId: "tiles",
      itemId: "it1",
      fieldId: "opacity",
      type: "NUMBER",
      value: 0.5,
      baseSceneRev: 9
    });
    expect(JSON.parse(s).d).toMatchObject({
      kind: "update_property_value",
      sceneId: "sc1",
      propertyId: "pr1",
      schemaGroupId: "tiles",
      itemId: "it1",
      fieldId: "opacity",
      type: "NUMBER",
      value: 0.5,
      baseSceneRev: 9
    });
  });

  it("builds NLS photo overlay apply envelopes", () => {
    const cr = applyCreateNlsPhotoOverlayPayload({
      sceneId: "sc1",
      layerId: "layp",
      baseSceneRev: 3
    });
    expect(JSON.parse(cr).d).toMatchObject({
      kind: "create_nls_photo_overlay",
      sceneId: "sc1",
      layerId: "layp",
      baseSceneRev: 3
    });
    const rm = applyRemoveNlsPhotoOverlayPayload({
      sceneId: "sc1",
      layerId: "layp",
      baseSceneRev: 4
    });
    expect(JSON.parse(rm).d).toMatchObject({
      kind: "remove_nls_photo_overlay",
      sceneId: "sc1",
      layerId: "layp",
      baseSceneRev: 4
    });
  });

  it("builds NLS sketch custom property apply envelopes", () => {
    const upd = applyUpdateNlsCustomPropertiesPayload({
      sceneId: "sc1",
      layerId: "layc",
      schema: { a: { type: "string" } },
      baseSceneRev: 1
    });
    expect(JSON.parse(upd).d).toMatchObject({
      kind: "update_nls_custom_properties",
      sceneId: "sc1",
      layerId: "layc",
      schema: { a: { type: "string" } },
      baseSceneRev: 1
    });
    const ch = applyChangeNlsCustomPropertyTitlePayload({
      sceneId: "sc1",
      layerId: "layc",
      schema: {},
      oldTitle: "x",
      newTitle: "y",
      baseSceneRev: 2
    });
    expect(JSON.parse(ch).d).toMatchObject({
      kind: "change_nls_custom_property_title",
      sceneId: "sc1",
      layerId: "layc",
      oldTitle: "x",
      newTitle: "y",
      baseSceneRev: 2
    });
    const rmProp = applyRemoveNlsCustomPropertyPayload({
      sceneId: "sc1",
      layerId: "layc",
      schema: {},
      removedTitle: "x",
      baseSceneRev: 3
    });
    expect(JSON.parse(rmProp).d).toMatchObject({
      kind: "remove_nls_custom_property",
      sceneId: "sc1",
      layerId: "layc",
      removedTitle: "x",
      baseSceneRev: 3
    });
  });

  it("builds NLS GeoJSON feature apply envelopes", () => {
    const add = applyAddNlsGeojsonFeaturePayload({
      sceneId: "sc1",
      layerId: "layg",
      type: "Feature",
      geometry: { type: "Point", pointCoordinates: [1, 2] },
      properties: { id: "f1" },
      baseSceneRev: 1
    });
    expect(JSON.parse(add).d).toMatchObject({
      kind: "add_nls_geojson_feature",
      sceneId: "sc1",
      layerId: "layg",
      type: "Feature",
      geometry: { type: "Point", pointCoordinates: [1, 2] },
      properties: { id: "f1" },
      baseSceneRev: 1
    });
    const upd = applyUpdateNlsGeojsonFeaturePayload({
      sceneId: "sc1",
      layerId: "layg",
      featureId: "fid",
      properties: { id: "f1", x: 1 },
      baseSceneRev: 2
    });
    expect(JSON.parse(upd).d).toMatchObject({
      kind: "update_nls_geojson_feature",
      sceneId: "sc1",
      layerId: "layg",
      featureId: "fid",
      properties: { id: "f1", x: 1 },
      baseSceneRev: 2
    });
    const del = applyDeleteNlsGeojsonFeaturePayload({
      sceneId: "sc1",
      layerId: "layg",
      featureId: "fid",
      baseSceneRev: 3
    });
    expect(JSON.parse(del).d).toMatchObject({
      kind: "delete_nls_geojson_feature",
      sceneId: "sc1",
      layerId: "layg",
      featureId: "fid",
      baseSceneRev: 3
    });
  });

  it("builds NLS infobox block apply envelopes", () => {
    const create = applyCreateNlsInfoboxPayload({
      sceneId: "sc1",
      layerId: "lay0",
      baseSceneRev: 0
    });
    expect(JSON.parse(create).d).toMatchObject({
      kind: "create_nls_infobox",
      sceneId: "sc1",
      layerId: "lay0",
      baseSceneRev: 0
    });

    const rmInf = applyRemoveNlsInfoboxPayload({
      sceneId: "sc1",
      layerId: "lay0",
      baseSceneRev: 1
    });
    expect(JSON.parse(rmInf).d).toMatchObject({
      kind: "remove_nls_infobox",
      sceneId: "sc1",
      layerId: "lay0",
      baseSceneRev: 1
    });

    const add = applyAddNlsInfoboxBlockPayload({
      sceneId: "sc1",
      layerId: "lay1",
      pluginId: "p~1",
      extensionId: "markdown",
      index: 0,
      baseSceneRev: 1
    });
    expect(JSON.parse(add).d).toMatchObject({
      kind: "add_nls_infobox_block",
      sceneId: "sc1",
      layerId: "lay1",
      pluginId: "p~1",
      extensionId: "markdown",
      index: 0,
      baseSceneRev: 1
    });

    const mv = applyMoveNlsInfoboxBlockPayload({
      sceneId: "sc1",
      layerId: "lay1",
      infoboxBlockId: "ib1",
      index: 2,
      baseSceneRev: 2
    });
    expect(JSON.parse(mv).d).toMatchObject({
      kind: "move_nls_infobox_block",
      sceneId: "sc1",
      layerId: "lay1",
      infoboxBlockId: "ib1",
      index: 2,
      baseSceneRev: 2
    });

    const rm = applyRemoveNlsInfoboxBlockPayload({
      sceneId: "sc1",
      layerId: "lay1",
      infoboxBlockId: "ib1",
      baseSceneRev: 3
    });
    expect(JSON.parse(rm).d).toMatchObject({
      kind: "remove_nls_infobox_block",
      sceneId: "sc1",
      layerId: "lay1",
      infoboxBlockId: "ib1",
      baseSceneRev: 3
    });
  });
});
