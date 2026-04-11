import { WidgetAlignSystemType } from "@reearth/services/gql";
import { describe, expect, it } from "vitest";

import {
  alignSystemForCollab,
  applyAddWidgetPayload,
  applyCreateStoryBlockPayload,
  applyMoveStoryBlockPayload,
  applyRemoveStoryBlockPayload,
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
});
