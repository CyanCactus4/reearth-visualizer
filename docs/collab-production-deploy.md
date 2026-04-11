# Collaboration — production deployment

This document complements [AGENTS.md](../AGENTS.md) and the [MVP design doc](design-doc/20260411_001_collaboration_protocol_mvp.md) for **operating** collab in production (multi-instance, persistence, security).

## Components

| Component | Role |
|-----------|------|
| **WebSocket** `GET /api/collab/ws` | Real-time room per `projectId`; JSON v1 protocol (`apply`, `chat`, `lock`, `cursor`, `activity`, …). |
| **Redis** `REEARTH_COLLAB_REDIS_URL` | Pub/sub between server instances + distributed object locks (Lua). Without Redis, relay and locks are **in-memory only** (single process). |
| **MongoDB** | Collections `collabChatMessages` (chat), `collabApplyAudit` (successful `apply` journal); names overridable via env (see below). |
| **SSE** `GET /api/collab/scene-rev/stream?sceneId=` | **Server-Sent Events** stream of `sceneRev` (scene `updatedAt` ms) after each successful collab `apply` on that scene (**in-process hub only**; other API replicas do not receive those events unless extended to Redis). Use when you want HTTP-based consumers without GraphQL subscriptions. |
| **REST** | `GET /api/collab/chat`, `GET /api/collab/apply-audit` — same auth as private `/api`. |

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
| `REEARTH_DB` / `REEARTH_DB_VIS` | Mongo connection and visualizer DB name (required for chat/audit stores). |

## Scaling

1. **≥2 API replicas:** set `REEARTH_COLLAB_REDIS_URL` so WS fan-out and locks are consistent across instances.
2. **Load balancers:** use **sticky sessions** for the collab WebSocket upgrade path if your LB does not guarantee same-node routing (optional when Redis relay is on; still helps for debugging).
3. **Mongo growth:** plan TTL or archival for `collabApplyAudit` and `collabChatMessages` in large deployments (not enforced in OSS).

## GraphQL subscriptions (optional)

The product today uses **Apollo over HTTP** for GraphQL plus a **separate collab WebSocket**. A native **GraphQL `Subscription`** for `sceneRev` would require:

- `gqlgen generate` with a `Subscription` root in `server/gql/*.graphql`,
- resolvers wired to the collab hub (or Redis) event bus,
- Apollo Client **split link** (`graphql-ws` or `subscriptions-transport-ws`) on the browser.

The **SSE** endpoint above is the supported lightweight alternative until GQL subscriptions are implemented.

## Security checklist (ops)

- TLS terminate at edge; WS and SSE use same origin policy as the web app.
- Never expose `/api/collab/*` without the same **JWT / session** middleware as GraphQL.
- Verify **project isolation** after deploy (see design doc “Production hardening”).
