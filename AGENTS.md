# AGENTS.md — контекст для ИИ-агента (Re:Earth Visualizer)

Этот документ — точка входа для агента в **корне** монорепозитория. Детальные правила сервера: [server/AGENTS.md](server/AGENTS.md). Гайд по фронту: [web/CLAUDE.md](web/CLAUDE.md). Шаблон дизайн-доков: [docs/design-doc-template.md](docs/design-doc-template.md).

## Техническое задание

Полное ТЗ на real-time collaboration — файл [TASK.md](TASK.md) (лабораторная + функциональные/технические требования). Любая новая работа по фиче должна явно трассироваться на пункты TASK.md.

## Окружение разработчика (типичное)

- **ОС в этой сессии:** Linux (desktop).
- **Стек:** Go (сервер), React 19 + TypeScript + Vite (web), MongoDB, Docker Compose для локального стенда, опционально Re:Earth Accounts + Cerbos (профиль `accounts` в [docker-compose.yml](docker-compose.yml)).
- **Фронт:** Node `>=20.11.0`, менеджер пакетов **Yarn 4.6.0** (Corepack).

## Структура репозитория

| Каталог | Назначение |
|--------|------------|
| `server/` | Бэкенд: Echo HTTP, GraphQL (gqlgen), gRPC internal API, Mongo-репозитории, GCS/S3, политики доступа |
| `web/` | SPA редактора и визуализатора: Apollo Client, Jotai, Cesium, фичи в `src/app/` |
| `e2e/` | Playwright: UI и API тесты |
| `cerbos/policies/` | Политики Cerbos (при режиме с accounts) |
| `docs/` | Документация и design docs |

## Архитектура данных (сервер)

Связи домена кратко (подробно — [server/README.md](server/README.md)):

- **Workspace** → много **Project**
- **Project** 1:1 **Scene** (на практике); **Scene** 1:1 **Story**
- **Scene** содержит: NLS-слои (в т.ч. группы), стили слоёв, виджеты, плагины, property сцены
- **Story** → страницы → блоки; конфигурация через общую систему **Property** / схем

Идентификаторы и операции сцен проходят через **usecase** и проверку **Operator** (читатель/писатель сцен и воркспейсов) — см. `server/internal/usecase/operator.go`.

## HTTP и GraphQL

- Приватный API: группа `/api` с middleware авторизации (`attachOpMiddlewareMockUser` или `attachOpMiddlewareReearthAccounts` в `server/internal/app/app.go`).
- GraphQL: **`POST /api/graphql`** (основной путь клиента). Playground в dev: **`GET /graphql`**.
- Совместное редактирование (MVP): **`GET /api/collab/ws?projectId=...`** — WebSocket после той же приватной авторизации, что и GraphQL; опционально **`REEARTH_COLLAB_REDIS_URL`** для pub/sub между инстансами и **для блокировок между инстансами** (Lua + `SET NX` в Redis; без URL — in-memory на процесс). Сообщения JSON v1: **`ping`**, **`relay`**, **`apply`** (`d.kind=update_widget` → `Scene.UpdateWidget` при праве на запись), **`presence`** (`join`/`leave`, `userId`), **`chat`**, **`cursor`** (`d.x`,`d.y` в [0,1], `inside`) → рассылка с `userId` и rate limit (**`REEARTH_COLLAB_CURSOR_MIN_INTERVAL_MS`**), **`activity`** (`d.kind` = `typing`|`move`, лимиты **`REEARTH_COLLAB_ACTIVITY_*`**), **`lock`** → **`lock_changed`** / **`lock_denied`**; TTL блокировок **`REEARTH_COLLAB_LOCK_TTL_SECONDS`**. Контекст: **`context.WithoutCancel`**. Клиент: `web/src/services/collab/` + **`CollabViewportCapture`** / **`CollabRemoteCursors`**, **`CollabPresenceBar`**, **`CollabLockGate`** / **`CollabLockReadOnly`** / **`CollabLockLeaseOnly`**: слой (lease на уровне вкладки Map при выборе), инспектор слоя без дублирующего lease, виджет, **контейнер виджетов (`widget_area`)**, **сцена (`scene`)**; при **`lock_denied`** — toast + модалка «конфликт» (MVP без merge); карта **`resourceLocks`** в контексте, offline **`offlineQueue`**, в редакторе передаётся **`localUserId`** из **`useMe`**.
- Обработчик gqlgen уже регистрирует `transport.Websocket` в `server/internal/app/graphql.go`, но в **схеме** (`server/gql/*.graphql`) **подписок (Subscription) сейчас нет** — real-time по ТЗ предстоит спроектировать (отдельный WS-протокол и/или GraphQL subscriptions + маршрутизация).
- Фронт: `web/src/services/gql/provider/index.tsx` — Apollo только с HTTP **upload** link, **без** `graphql-ws` / split для подписок.

## Авторизация

- **Mock auth:** флаг mock в конфиге; контекст `adapter.AttachMockAuth`, демо-пользователь.
- **Prod-like:** токены через Re:Earth Accounts; сцены/проекты фильтруются по правам оператора.

Любой новый WS или subscription обязан повторять те же проверки, что и мутации GraphQL (проект/сцена/workspace).

## Тесты и качество

- **Сервер:** `cd server && make test`, `make e2e`, `make lint`; юнит-тесты рядом с кодом (`*_test.go`), e2e в `server/e2e/`.
- **Web:** `cd web && yarn test`, `yarn type`, `yarn lint`, при смене схемы — `yarn gql`.
- **E2E:** `e2e/` — Playwright; для локального прогона см. скрипты в `e2e/package.json`.

**Правило:** не ломать существующее поведение; для новых фич — новые тесты (юнит + по возможности e2e).

## Git: коммит после каждой функциональности

**Требование к разработке:** по завершении логически целого куска работы (одна функциональность, одна фича из плана) агент **создаёт коммит** в git: сначала прогоняет затронутые тесты/линтер, затем `git add` только нужные файлы и `git commit` с сообщением в стиле **Conventional Commits** и scope (`feat(server):`, `fix(web):`, `test(server):`, …), как в [server/AGENTS.md](server/AGENTS.md). Не смешивать в одном коммите несвязанные изменения; не коммитить секреты и локальные `.env`. Пуш на `origin` — только если у пользователя в allowlist есть `git push` (см. [COMMANDS.md](COMMANDS.md)).

## Coding guidelines (кратко)

- **Go:** как в [server/AGENTS.md](server/AGENTS.md) — gqlgen, `make gql` после изменения схемы, табы, golangci-lint.
- **TS/React:** стиль проекта (ESLint `eslint-config-reearth`), Jotai для состояния редактора, Apollo для GQL.
- **Объём изменений:** только необходимое для задачи; без рефакторинга «заодно».

## Что важно помнить для TASK.md (collaboration)

- Collab MVP в коде: **gorilla/websocket**, опциональный **Redis** (relay + locks), без OT/CRDT для сцены — дальнейшее по [PLAN.md](PLAN.md) / [TASK.md](TASK.md).
- История undo/redo в смысле **всего проекта** — не как единая готовая подсистема; локальный undo есть у отдельных UI (например Lexical). Мультипользовательская история из ТЗ потребует новой модели событий и хранения (Mongo).

## Обновление этого файла

Дополняйте AGENTS.md, если агент регулярно «теряется» на одном и том же шаге (конкретная команда, переменная окружения, порядок запуска Docker).
