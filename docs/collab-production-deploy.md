# Collaboration — production deployment

This document complements [AGENTS.md](../AGENTS.md) and the [MVP design doc](design-doc/20260411_001_collaboration_protocol_mvp.md) for **operating** collab in production (multi-instance, persistence, security).

## Components

| Component | Role |
|-----------|------|
| **WebSocket** `GET /api/collab/ws` | Real-time room per `projectId`; JSON v1 protocol (`apply`, `chat`, `lock`, `cursor`, `activity`, …). |
| **Redis** `REEARTH_COLLAB_REDIS_URL` | Pub/sub between server instances + distributed object locks (Lua). Without Redis, relay and locks are **in-memory only** (single process). |
| **MongoDB** | Collections `collabChatMessages` (chat), `collabApplyAudit` (successful `apply` journal), `collabUndoOps` + `collabUndoState` (server undo/redo stacks); names overridable via env (see below). |
| **SSE** `GET /api/collab/scene-rev/stream?sceneId=` | **Server-Sent Events** stream of `sceneRev` (scene `updatedAt` ms) after each successful collab `apply` on that scene. With **`REEARTH_COLLAB_REDIS_URL`**, revisions are also **fan-out via Redis** (`collab:srev:<sceneId>`) so every API instance updates its local subscribers. |
| **GraphQL** `POST/GET /api/graphql` | Standard queries/mutations over HTTP; **WebSocket upgrade on GET** serves `graphql-ws` / `graphql-transport-ws` for **`subscription { collabSceneRevision(sceneId) }`** (scene `updatedAt` ms after collab applies). Same Redis fan-out as SSE when Redis is enabled. |
| **REST** | `GET /api/collab/chat`, `GET /api/collab/apply-audit`, `POST /api/collab/undo`, `POST /api/collab/redo` — same auth as private `/api`. |

## Environment variables (collab-related)

| Variable | Purpose |
|----------|---------|
| `REEARTH_COLLAB_REDIS_URL` | Redis for relay + locks. |
| `REEARTH_COLLAB_LOCK_TTL_SECONDS` | Lock inactivity timeout (default 300). |
| `REEARTH_COLLAB_MAX_MESSAGE_BYTES` | Max WS frame size. |
| `REEARTH_COLLAB_MAX_MESSAGES_PER_SEC` | Per-connection rate limit. |
| `REEARTH_COLLAB_CHAT_MAX_RUNES` / `REEARTH_COLLAB_CHAT_MIN_INTERVAL_MS` | Chat size and per-user spacing. |
| `REEARTH_COLLAB_CHAT_COLLECTION` | Mongo chat collection (default `collabChatMessages`). |
| `REEARTH_COLLAB_APPLY_AUDIT_COLLECTION` | Mongo apply audit collection (default `collabApplyAudit`). |
| `REEARTH_COLLAB_OPLOG_COLLECTION` | Mongo undo op log (default `collabUndoOps`). |
| `REEARTH_COLLAB_UNDO_STATE_COLLECTION` | Mongo per-user undo/redo stacks (default `collabUndoState`). |
| `REEARTH_COLLAB_MENTION_WEBHOOK_URL` | Optional HTTPS URL: POST JSON on each `@mention` (in addition to in-room WS `notify`). |
| `REEARTH_DB` / `REEARTH_DB_VIS` | Mongo connection and visualizer DB name (required for chat/audit/undo stores). |

## Scaling

1. **≥2 API replicas:** set `REEARTH_COLLAB_REDIS_URL` so WS fan-out and locks are consistent across instances.
2. **Load balancers:** use **sticky sessions** for the collab WebSocket upgrade path if your LB does not guarantee same-node routing (optional when Redis relay is on; still helps for debugging).
3. **Mongo growth:** plan TTL or archival for `collabApplyAudit` and `collabChatMessages` in large deployments (not enforced in OSS).

## GraphQL subscriptions

- **Server:** `subscription { collabSceneRevision(sceneId: ID!): Int! }` — resolver reads the collab hub (`AttachHub` on GraphQL context). **Run `make gql`** in `server/` after schema edits so `generated.go` stays in sync (this repo may ship a pre-patched `generated.go` when codegen cannot reach the network).
- **Web:** Apollo uses a **split link** (`graphql-ws` + `GET /api/graphql` WebSocket). The editor `CollabProvider` subscribes when `sceneId` is set and refetches `GetScene` on each revision.
- **SSE** remains a simple HTTP alternative for non-Apollo consumers.

## Entity clocks (LWW) and undo

- **Per-widget field clocks** (`enabled`, `extended`, `layout`): with **`REEARTH_COLLAB_REDIS_URL`**, values live under Redis keys `collab:wfclk:…` (atomic `INCR`) so **all API replicas share the same LWW sequence**. Without Redis, clocks are **in-memory only** on each process (restart clears them).
- **Undo/redo** applies inverse JSON through the same interactors as collab; stacks live in Mongo (`collabUndoOps` / `collabUndoState`). Recorded kinds today: **`update_widget`**, **`move_story_block`**, **`move_story_page`**, **`update_story_page`**, **`update_property_value`**, **`update_style`**. Add/remove widget are **not** on the undo stack (redo would not preserve widget identity without a dedicated restore path). `ExecuteCollabUndoJSON` checks **`IsWritableScene`** for every supported kind (defense in depth alongside `POST /api/collab/undo|redo`).
- **Not** a distributed CRDT log — plan stronger consistency for multi-writer undo if needed.

## Security checklist (ops)

- TLS terminate at edge; WS and SSE use same origin policy as the web app.
- Never expose `/api/collab/*` without the same **JWT / session** middleware as GraphQL.
- Verify **project isolation** after deploy (see design doc “Production hardening”).
