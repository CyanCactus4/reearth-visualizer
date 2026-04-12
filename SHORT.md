# SHORT.md — конспект для защиты проекта (Re:Earth Visualizer + real-time collaboration)

Документ описывает **что это за система**, **из каких частей она состоит**, **какие технологии используются**, **где в коде лежит реализованный функционал совместного редактирования** и **что осознанно не входит в текущий MVP**. Формулировки привязаны к файлам и каталогам репозитория; при расхождении с кодом приоритет у кода.

---

## 1. Назначение продукта

**Re:Earth Visualizer** — веб-приложение для работы с геоданными: сцена (карта на Cesium), слои (NLS), стили слоёв, виджеты, плагины, storytelling (страницы и блоки), публикация проектов. Данные и права доступа проходят через **бэкенд на Go** (HTTP API + GraphQL + часть internal gRPC) и **MongoDB** как основное хранилище доменных сущностей.

В рамках варианта по **TASK.md** в репозитории реализован **модуль real-time collaboration (collab)**: несколько клиентов с доступом к одному проекту могут одновременно редактировать сцену с серверной валидацией, рассылкой событий по WebSocket, опциональной синхронизацией между инстансами API через **Redis**, журналированием и **server-side undo/redo** для части операций.

---

## 2. Технологический стек (фиксированный перечень)

| Слой | Технология | Где видно в репозитории |
|------|------------|-------------------------|
| Бэкенд | Go (модули `server/go.mod`) | [server/](server/) |
| HTTP / middleware | labstack/echo | [server/internal/app/](server/internal/app/) |
| GraphQL | gqlgen | [server/gql/](server/gql/), кодогенерация: `cd server && make gql` |
| WebSocket collab | gorilla/websocket | [server/internal/collab/ws.go](server/internal/collab/ws.go), регистрация маршрута в приложении |
| Фронтенд | React 19, TypeScript, Vite | [web/package.json](web/package.json) |
| Состояние UI редактора | Jotai | атомы и провайдеры в [web/src/app/features/Editor/](web/src/app/features/Editor/) |
| Запросы к API | Apollo Client; для подписок — `graphql-ws` (split link) | [web/src/services/gql/](web/src/services/gql/) |
| Карта | Cesium (через `@reearth/core` и визуализатор) | [web/src/app/features/Visualizer/](web/src/app/features/Visualizer/) |
| БД | MongoDB | репозитории в [server/internal/infrastructure/mongo/](server/internal/infrastructure/mongo/) |
| Локальный стенд | Docker Compose | корневой [docker-compose.yml](docker-compose.yml), цели `make d-run` в [server/Makefile](server/Makefile) |
| Юнит-тесты сервера | `testing`, stretchr/testify | файлы `*_test.go` рядом с кодом |
| Юнит-тесты фронта | Vitest | [web/package.json](web/package.json) скрипт `test` |
| E2E (отдельный пакет) | Playwright | [e2e/](e2e/) |

**Node:** в [web/package.json](web/package.json) указано `engines.node`: `>=20.11.0`. **Yarn:** в корне монорепозитория используется Yarn 4 (Corepack); точная политика версии — в [AGENTS.md](AGENTS.md).

---

## 3. Структура монорепозитория (что за что отвечает)

| Каталог | Ответственность |
|---------|-----------------|
| [server/](server/) | API, GraphQL, collab WS/REST, interactors, Mongo, конфигурация |
| [web/](web/) | SPA редактора и визуализатора |
| [e2e/](e2e/) | Playwright: UI-сценарии и API-сценарии (в т.ч. два WebSocket-клиента для collab) |
| [docs/](docs/) | Эксплуатационные и дизайн-документы (в т.ч. [docs/collab-production-deploy.md](docs/collab-production-deploy.md)) |
| [openspec/](openspec/) | OpenSpec (SDD), не исполняется рантаймом приложения |
| [TASK.md](TASK.md) | Текст задания и **статус реализации** в репозитории |
| [PLAN.md](PLAN.md) | План фаз и формальное закрытие MVP по фазам |
| [AGENTS.md](AGENTS.md) | Инструкции для разработчика/агента: эндпоинты collab, тесты, окружение |

---

## 4. Доменная модель (минимум для понимания collab)

- **Workspace** содержит много **Project**.
- У **Project** есть **Scene** (практически 1:1).
- **Scene** содержит: NLS-слои, стили, виджеты, плагины, **Property** сцены (JSON-подобная конфигурация по схеме плагина).
- **Story** привязан к сцене; **StoryPage** / **StoryBlock** — storytelling.
- Любая операция записи на сервере проходит через **usecase** и проверку **Operator** (читатель/писатель сцены и воркспейса). Collab **не обходит** эти проверки: операции вызывают те же interactors, что и GraphQL-мутации, где это так задумано в коде.

Подробная ER-диаграмма в [server/README.md](server/README.md).

---

## 5. Архитектура collaboration (поток данных)

### 5.1. Транспорты и эндпоинты

1. **WebSocket collab:** `GET /api/collab/ws?projectId=<id>`  
   - Аутентификация: **тот же механизм**, что и для приватной группы `/api` (JWT / mock — в зависимости от конфигурации сервера).  
   - Протокол сообщений: JSON с полем версии и типа верхнего уровня (см. обработчик в [server/internal/collab/ws.go](server/internal/collab/ws.go)).

2. **Тело операции редактирования:** клиент отправляет кадр с `t: "apply"` и полем `d`, внутри `d` обязательно **`kind`** (строка). Сервер маршрутизирует по `kind` в [server/internal/collab/apply.go](server/internal/collab/apply.go) → `dispatchApply`.

3. **Уведомление о новой ревизии сцены:**  
   - GraphQL: `subscription { collabSceneRevision(sceneId: ID!): Int! }`  
   - Дополнительно: SSE `GET /api/collab/scene-rev/stream?sceneId=...`  
   Значение — монотонная метка, производная от `scene.UpdatedAt` в миллисекундах (как задокументировано в [AGENTS.md](AGENTS.md)). При наличии **Redis** (`REEARTH_COLLAB_REDIS_URL`) обновления **sceneRev** дублируются между инстансами API.

4. **Чат и журнал:** REST `GET /api/collab/chat`, `GET /api/collab/apply-audit`; запись чата и аудита — в Mongo (имена коллекций и переменные окружения — [docs/collab-production-deploy.md](docs/collab-production-deploy.md)).

5. **Undo / redo на сервере:** `POST /api/collab/undo`, `POST /api/collab/redo` с телом `{"sceneId":"<id>"}`; исполнение — [server/internal/collab/undo_exec.go](server/internal/collab/undo_exec.go).

### 5.2. Компоненты сервера collab

| Компонент | Файлы / пакет | Роль |
|-----------|---------------|------|
| Hub комнат | [server/internal/collab/hub.go](server/internal/collab/hub.go) | Регистрация соединений по `projectId`, broadcast в комнату |
| Соединение | [server/internal/collab/conn.go](server/internal/collab/conn.go) | Контекст пользователя, сцена комнаты, очередь исходящих |
| WS read/write | [server/internal/collab/ws.go](server/internal/collab/ws.go) | Upgrade, лимиты размера и частоты сообщений |
| Redis relay | [server/internal/collab/redis_relay.go](server/internal/collab/redis_relay.go) | Pub/sub между процессами API |
| Блокировки объектов | [server/internal/collab/lock_*.go](server/internal/collab/) | Ресурсы `layer`, `widget`, `scene`, `widget_area`, `style`; при Redis — распределённые блокировки |
| Активность / курсор | [server/internal/collab/activity_dispatch.go](server/internal/collab/activity_dispatch.go), [cursor_dispatch.go](server/internal/collab/cursor_dispatch.go) | `typing`, `move`; координаты курсора |
| Property HLC / часы | [server/internal/collab/hlc.go](server/internal/collab/hlc.go), [property_field_hlc.go](server/internal/collab/property_field_hlc.go) | LWW-регистры на поля property с гибридными логическими часами |

### 5.3. Перечень `d.kind` для сообщения `apply` (на момент описания)

Значения обрабатываются в [server/internal/collab/apply.go](server/internal/collab/apply.go). Любая другая строка → ответ сервера с кодом **`unknown_kind`** (см. ветку `default` в `dispatchApply`).

Список поддерживаемых `kind`:

`update_widget`, `remove_widget`, `add_widget`, `move_story_block`, `create_story_block`, `remove_story_block`, `create_story_page`, `remove_story_page`, `move_story_page`, `update_story_page`, `duplicate_story_page`, `add_nls_layer_simple`, `remove_nls_layer`, `update_nls_layer`, `update_nls_layers`, `create_nls_infobox`, `remove_nls_infobox`, `create_nls_photo_overlay`, `remove_nls_photo_overlay`, `add_nls_infobox_block`, `move_nls_infobox_block`, `remove_nls_infobox_block`, `update_nls_custom_properties`, `change_nls_custom_property_title`, `remove_nls_custom_property`, `add_nls_geojson_feature`, `update_nls_geojson_feature`, `delete_nls_geojson_feature`, `add_style`, `update_style`, `remove_style`, **`update_scene_camera`** (отдельный вход: сервер сам находит корневой `propertyId` и элемент группы `camera`, затем применяет тот же путь, что и `update_property_value`), `update_property_value`, `merge_property_json`, `add_property_item`, `remove_property_item`, `move_property_item`.

### 5.4. Синхронизация и «CRDT» в смысле MVP

- **Не** реализован полный документный CRDT уровня **Yjs/Automerge** на всё дерево property как единый автомат.
- **Реализовано явно:**  
  - для полей property — режим **LWW + Hybrid Logical Clock** (`fieldHlc` в `update_property_value`) и совместимость с целочисленным **`fieldClock`**;  
  - **`merge_property_json`** — merge-patch по плоским ключам листьев с **CAS `docClock`** на документ property;  
  - операции над элементами списков property — `add_property_item`, `remove_property_item`, `move_property_item`;  
  - для виджетов — **`entityClocks`** (per-field LWW) в `update_widget`.

Это зафиксировано в [TASK.md](TASK.md) (раздел «Статус реализации») и не отменяет требование ТЗ «OT или CRDT» в формулировке «допускается класс решений с серверной нормализацией и LWW-регистрами», но **не** тождественно «весь JSON через Yjs».

### 5.5. Клиентский слой collab

| Назначение | Расположение |
|------------|----------------|
| Провайдер WS, подписка `collabSceneRevision`, чат, курсоры, activity, offline queue | [web/src/services/collab/CollabProvider.tsx](web/src/services/collab/CollabProvider.tsx) |
| Контекст и типы | [web/src/services/collab/collabContext.ts](web/src/services/collab/collabContext.ts) |
| Сборка JSON кадров `apply` | [web/src/services/collab/applyMessages.ts](web/src/services/collab/applyMessages.ts) |
| Маршрутизация мутаций на collab при открытом сокете | хуки в [web/src/services/api/](web/src/services/api/) (например `usePropertyMutations`, `useWidgetMutations`, NLS, story, …) |
| UI: курсоры, presence, чат, история apply, блокировки | [web/src/app/features/Editor/](web/src/app/features/Editor/) (`CollabRemoteCursors`, `CollabPresenceBar`, `CollabChatPanel`, `CollabApplyHistoryPanel`, `CollabLockGate`, …) |

Редактор оборачивается в `CollabProvider` при наличии `projectId` (см. [web/src/app/features/Editor/index.tsx](web/src/app/features/Editor/index.tsx)).

### 5.6. Блокировки и конфликты

- Клиент запрашивает блокировку через WS-сообщение типа **`lock`** (см. [AGENTS.md](AGENTS.md)).  
- При отказе приходит **`lock_denied`**; в UI показываются toast и модалка сценария конфликта (сравнение снимков сцены — см. реализацию модалки и колбэки в `CollabProvider`).  
- TTL блокировок задаётся переменной **`REEARTH_COLLAB_LOCK_TTL_SECONDS`** (по умолчанию 300 секунд — см. [docs/collab-production-deploy.md](docs/collab-production-deploy.md)).

---

## 6. Что не следует выдавать за реализованное «дословно по всему ТЗ»

Ниже — **явные границы**, чтобы на защите не было ложных утверждений:

1. **Приглашение пользователей в проект** как продуктовая фича workspace/ACL — не ядро collab-модуля; collab предполагает, что доступ к проекту уже выдан.
2. **Yjs/Automerge на всё дерево property** — вне текущего объёма; см. [TASK.md](TASK.md).
3. **Отдельный collab-`kind` на каждое поле настроек сцены** (terrain, sky, lighting, …): кроме **`update_scene_camera`** остальное идёт через **`update_property_value` / `merge_property_json`** и обновление сцены через refetch / `sceneRev`.
4. **Лабораторные пункты** про OpenCode/OpenSpec-плагин и «процесс Plan/Build» — частично вне git; артефакты OpenSpec под Cursor — в [openspec/](openspec/) и [.cursor/commands/](.cursor/commands/).

---

## 7. Документы для углубления перед вопросами комиссии

- [TASK.md](TASK.md) — требования и статус.  
- [PLAN.md](PLAN.md) — фазы и формальное закрытие MVP.  
- [AGENTS.md](AGENTS.md) — полный перечень collab URL и сообщений.  
- [docs/design-doc/20260411_001_collaboration_protocol_mvp.md](docs/design-doc/20260411_001_collaboration_protocol_mvp.md) — протокол MVP.  
- [docs/collab-production-deploy.md](docs/collab-production-deploy.md) — прод: Redis, Mongo-коллекции, переменные окружения.

---

## 8. Связь с файлами запуска и тестов

Пошаговый запуск локального стенда — **[START.md](START.md)**.  
Полный перечень команд тестирования — **[TEST.md](TEST.md)**.
