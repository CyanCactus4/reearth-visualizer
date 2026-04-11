/**
 * Server `error.d.code` after a failed collab `apply` (or lock lookup for widget apply).
 * When received, Apollo GetScene should refetch so local UI matches server (PLAN phase 3).
 */
export const COLLAB_APPLY_FAILURE_SCENE_REFETCH_CODES = new Set<string>([
  "apply_failed",
  "object_locked",
  "stale_state",
  "stale_entity_field",
  "invalid_payload",
  "invalid_scene",
  "scene_mismatch",
  "invalid_widget",
  "invalid_align",
  "empty_update",
  "invalid_location",
  "invalid_plugin",
  "invalid_extension",
  "invalid_story",
  "invalid_page",
  "invalid_block",
  "invalid_layer",
  "invalid_layer_type",
  "unknown_kind",
  "lock_lookup",
  "internal"
]);
