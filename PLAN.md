# PLAN.md — план реализации ТЗ (TASK.md) на 100%

Документ привязан к [TASK.md](TASK.md): real-time collaboration для **reearth/reearth-visualizer**. Порядок фаз выбран так, чтобы сначала появилась безопасная инфраструктура, затем минимально полезный продуктовый срез, затем усложнение (OT/CRDT, offline, админский откат).

Принципы: **не ломать** текущие GraphQL-мутации и публичные сценарии; каждая фаза заканчивается зелёными тестами; новые публичные поверхности сопровождаются тестами и при необходимости фичефлагом. После каждой **завершённой функциональности** (осмысленный инкремент в рамках фазы) — **отдельный git-коммит** с Conventional Commits (см. [COMMANDS.md](COMMANDS.md) и [AGENTS.md](AGENTS.md)), чтобы можно было откатиться точечно.

---

## Фаза 0 — Проектирование и контракты (без изменения поведения для пользователей)

**Цель:** зафиксировать протокол и границы ответственности.

1. **Выбор транспорта real-time**
   - Вариант A: отдельный WebSocket endpoint (например `/api/collab` или `/ws`) на Echo + `gorilla/websocket` — полный контроль сообщений, проще лимиты и кастомная валидация (как в ТЗ).
   - Вариант B: GraphQL Subscriptions через gqlgen (уже есть `transport.Websocket` в handler) — единый стек с клиентом, но нужны **GET** (или явный WS) на том же пути, что и Apollo, и договорённость о протоколе `graphql-transport-ws` / `graphql-ws`.
   - **Рекомендация для ТЗ:** гибрид — доменный collab и чат через **отдельный WS**; опционально позже тонкие «уведомления» дублировать в GQL subscription, если появится потребность.

2. **Формат сообщений (envelope):** `type`, `roomId` (projectId), `seq`, `clientId`, `payload`, `ack`/`error` коды; максимальный размер тела; версия протокола.

3. **Модель синхронизации:** зафиксировать **CRDT vs OT** по сущностям:
   - Для иерархий слоёв/виджетов/страниц историй чаще подходят **операции поверх дерева** + серверная нормализация или CRDT для JSON-подобных property (оценить библиотеки и размер состояния сцены).
   - ТЗ допускает OT **или** CRDT — выбрать один основной подход и документировать исключения.

4. **Персистентность истории:** схема коллекций Mongo (событие, автор, projectId, timestamp, undo-group для «своих» действий).

5. **Security checklist:** JWT из заголовка/первого сообщения; проверка membership в project/workspace теми же слоями, что и в interactors; rate limits; размер сообщений.

**Выход:** короткий design doc в `docs/design-doc/` (или дополнение к существующему процессу), диаграмма последовательности join room → sync → broadcast.

---

## Фаза 1 — Инфраструктура WebSocket и комнаты (FR-1)

**Бэкенд**

1. Поднять **WS hub** (goroutine per room или shard map + mutex): join/leave, broadcast в комнату «проект».
2. **Room = projectId** (или sceneId, если доступ строго к сцене — согласовать с моделью прав `Operator`).
3. **Аутентификация:** при `Upgrade` или первом сообщении передать JWT; валидация через существующий путь accounts/mock; в контекст класть operator/user id.
4. **Redis Pub/Sub:** публикация событий комнаты в канал `collab:{projectId}`; подписка каждого инстанса; локальный hub рассылает подключённым клиентам. Конфиг Redis через env (`REEARTH_REDIS_*` или аналог).
5. **Валидация входящих сообщений:** JSON schema / структуры Go + отклонение неизвестных `type`.
6. **DoS-защита:** лимит размера кадра WS, лимит сообщений/сек на соединение, таймаут idle.

**Тесты:** юнит-тесты hub + интеграционный тест с in-memory Redis или testcontainers (по возможности репозитория).

**Фронт (минимум):** заготовка модуля `CollabSocket` (connect, reconnect, exponential backoff) **без** ещё изменения редакторов.

---

## Фаза 2 — Синхронизация состояния: MVP по одной сущности (FR-2, частично)

**Цель:** доказать сквозной поток «изменение → сервер → другие клиенты» на **одной** сущности с низким риском (например **widgets** или **layer order**), затем расширять.

1. Определить **операции** (add/update/remove/move) и атомарность на уровне одного сообщения.
2. Сервер: применение операции к доменной модели через существующие **interactors** / репозитории (не дублировать бизнес-правила вне usecase).
3. Сохранение в Mongo в той же транзакции/порядке, что и сегодня для мутаций (где возможно).
4. **Клиент:** буфер исходящих при обрыве; при reconnect — **snapshot** или догоняющий diff (версия сцены `rev` в проекте/сцене).

**Тесты:** серверные тесты операций; фронт — Vitest на reducer/merge; e2e по желанию два вкладки Playwright.

Расширить на: **layers / layer styles / scene settings / widgets / storytelling** — итерациями по приоритету продукта, с общим каркасом операций.

**Срез (NLS):** collab-`apply` с `d.kind` **`add_nls_layer_simple`**, **`remove_nls_layer`**, **`update_nls_layer`**, **`update_nls_layers`** (перестановка порядка); клиент — `useNLSLayerMutations(sceneId)` и хелперы в `web/src/services/collab/applyMessages.ts`.

**Срез (стили слоёв / `Style`):** **`add_style`**, **`update_style`**, **`remove_style`**; клиент — `useLayerStyleMutations(sceneId)`; блокировка **`resource: "style"`** на сервере и в типе `LockResource`.

---

## Фаза 3 — OT/CRDT и атомарность для всех сущностей из ТЗ (FR-2)

1. Выровнять модель конфликтов с **блокировками** (фаза 4): операции либо принимаются, либо отклоняются с кодом «object locked».
2. Реализовать **откат** неуспешной операции на клиенте (optimistic UI) по ответу сервера.
3. Покрыть **storytelling** (страницы, блоки) отдельным набором операций из-за связей с плагинами и property.

**Тесты:** табличные тесты трансформации/мерджа; регрессия на больших JSON property.

---

## Фаза 4 — Блокировки и конфликты (FR-4)

1. **Layer-level (и при необходимости widget/block):** при `edit:start` сервер резервирует объект для userId, рассылает `lock:granted` / `lock:denied`.
2. **Таймаут 5 минут** бездействия — снятие блокировки (heartbeat от клиента или серверный таймер по last activity).
3. **Одновременное редактирование:** диалог выбора версии на клиенте по событию `conflict` с двумя snapshot/diff.

**Тесты:** таймауты с укороченным интервалом в тестовом конфиге; гонка двух клиентов.

---

## Фаза 5 — Визуальные индикаторы присутствия (FR-3)

1. Протокол **presence:** список пользователей в комнате, heartbeat, цвет/имя.
2. **Курсоры** (lat/lng или экранные координаты — согласовать с Cesium): throttle на клиенте.
3. **Подсветка редактируемого объекта** данными из lock/presence.
4. **«Печатает» / «перемещает»** — лёгкие presence-события.

**Фронт:** React context/provider для collab state; компоненты UI в зоне редактора (см. `web/src/app/features/Editor/`).

**Тесты:** компонентные тесты + при необходимости визуальные сценарии в e2e.

---

## Фаза 6 — История, undo/redo, админский откат (FR-5)

1. **Журнал операций** в Mongo с автором.
2. **Undo/redo только своих** действий: клиент хранит локальную стековую группу, сервер откатывает по обратной операции или по записи «compensating op».
3. **UI истории** с автором.
4. **Администратор проекта:** мутация «restore project to revision» с проверкой роли maintainer/owner.

**Тесты:** цепочки undo/redo в многопользовательской симуляции; права на restore.

---

## Фаза 7 — Уведомления и чат (FR-6 + техтребования)

1. Типы WS-сообщений: `chat:message`, `chat:typing`, `notify:action`.
2. **Чат** с привязкой к project room; персистентность сообщений в Mongo (с пагинацией).
3. **@mentions:** парсинг и подписка упомянутых пользователей (уведомление).
4. **Rate limiting** чата на сервере (token bucket по user+project).

**Тесты:** нагрузочный лимит; валидация длины сообщения.

---

## Фаза 8 — GraphQL расширения (техтребования бэкенда)

1. Новые **мутации** (например управление блокировками, restore revision, отправка чата — если не только WS).
2. **Подписки** (если выбраны): добавить в `server/gql/*.graphql`, реализовать resolvers, **зарегистрировать GET/WebSocket** на Echo для совместимости с клиентом подписок.
3. Документация для фронта и примеры запросов.

---

## Фаза 9 — Фронтенд: Apollo, offline, read-only режим (техтребования фронта)

1. Интеграция WS с **Apollo** (либо parallel channel: GQL для запросов/мутаций, WS для collab — допустимо и проще).
2. **Collaboration provider** вокруг редактора.
3. **Модификация редакторов** для read-only при lock другого пользователя (`web/src/app/features/Editor/`, инспекторы, storytelling).
4. **Graceful degradation / offline:** очередь операций в IndexedDB/localforage (уже есть `localforage` в зависимостях), синхронизация после online.

**Тесты:** интеграционные тесты очереди; ручной чеклист Cesium.

---

## Фаза 10 — Закрытие ТЗ и hardening

1. Пентест-чеклист: права, утечки комнат между проектами, подделка projectId.
2. Метрики и логи (OpenTelemetry уже в сервере).
3. Документация деплоя: Redis, масштабирование инстансов, переменные окружения.

---

## Трассировка TASK.md → фазы

| Блок TASK.md | Фазы |
|--------------|------|
| FR-1 WebSocket, rooms, JWT, Redis | 0–1 |
| FR-2 синхронизация сущностей, атомарность, буфер | 0, 2–3 |
| FR-3 presence, курсоры, индикаторы | 5 (+ частично 4) |
| FR-4 блокировки, таймаут, конфликт UI | 4 |
| FR-5 история, undo/redo, авторы, админ restore | 6 |
| FR-6 уведомления, чат, @ | 7 |
| GraphQL мутации/подписки, WS hub, Redis, Mongo история, rate limit | 1, 6–8 |
| Фронт: Apollo/WS, provider, UI, read-only, offline | 2, 5, 9 |
| Безопасность и DoS | 0–1, 7, 10 |

---

## Рекомендации по SDD / OpenSpec (из TASK.md)

После стабилизации контрактов фазы 0 выполнить цикл **explore → propose → apply → verify → archive** в выбранном инструменте (OpenSpec и т.д.), чтобы изменения шли спецификацией, а не только кодом.

---

## Статус фаз (живой ориентир)

Следующая по приоритету **незакрытая** фаза после последнего коммита: смотри строку «→ дальше». Детальный контракт фазы 0: [docs/design-doc/20260411_001_collaboration_protocol_mvp.md](docs/design-doc/20260411_001_collaboration_protocol_mvp.md).

| Фаза | Статус | Комментарий |
|------|--------|-------------|
| 0 Проектирование и контракты | ✅ | Design doc выше; транспорт и v1-протокол зафиксированы. |
| 1 WS и комнаты | ✅ | Hub, `projectId`, JWT, Redis relay, лимиты. |
| 2 MVP-синхронизация одной сущности | ✅ | `apply`: виджеты + story **blocks** + story **pages**; **NLS-слои**; **стили слоя (`Style`)**; **`update_property_value`** (значения полей property через interactors); `applied` + **`sceneRev`**; Vitest `applyMessages`; фронт: **`useStoryPageMutations(sceneId)`**, **`useStoryBlockMutations(sceneId)`**, **`useWidgetMutations`**, **`useNLSLayerMutations(sceneId)`**, **`useLayerStyleMutations(sceneId)`**, **`usePropertyMutations(sceneId)`** там, где в инспектор передан **`sceneId`**. |
| 3 OT/CRDT | 🟡 | **`baseSceneRev`** + `stale_state`; storytelling blocks + **pages** (в т.ч. duplicate); **LWW `entityClocks`** (+ Redis при `REEARTH_COLLAB_REDIS_URL`); NLS-слои и **стили слоя (`Style`)** через `apply`. **→ дальше:** CRDT/merge по JSON property; **`sceneId`** в редакторе доходит до plugin blocks (story/infobox) → `PropertyItem`; остаётся дожать редкие обходы без `sceneId` и прочие NLS-подоперации. |
| 4 Блокировки | 🟡 | Locks + UI + `apply`; модалка: reload + **сравнение двух снимков** (кэш Apollo vs network: счётчики widgets/stories). Полный трёхсторонний merge — вне scope. |
| 5 Presence | 🟡 | Курсоры (в т.ч. **`title` = полный userId**), typing, полоса presence; без отдельного аватара на курсоре. |
| 6 История / undo | 🟡 | Mongo + REST + **`CollabApplyHistoryPanel`**; **серверный** undo/redo (`POST /api/collab/undo|redo`) для **`update_widget`**, **`move_story_block`**, **`move_story_page`**, **`update_story_page`**, **`update_property_value`** и **`update_style`**; **нет** undo add/remove виджета и **нет** админского restore ревизии. |
| 7 Уведомления и чат | 🟡 | Чат + Mongo + toasts по `applied`; **`@mentions`**: парсинг, Mongo/WS, подсветка; **WS `notify`** `chat_mention`; опционально **webhook** `REEARTH_COLLAB_MENTION_WEBHOOK_URL`. |
| 8 GraphQL | 🟡 | **`subscription { collabSceneRevision(sceneId) }`**, GET+POST `/api/graphql` + WebSocket; **SSE** scene-rev; fan-out rev по Redis между инстансами. См. **`docs/collab-production-deploy.md`**. |
| 9 Фронт Apollo/offline | 🟡 | Provider + **`localforage`** offline queue (`offlineQueue.ts`); reconcile сцены из редактора при lock-conflict. |
| 10 Hardening | 🟡 | Чеклист в design doc + **[docs/collab-production-deploy.md](docs/collab-production-deploy.md)** (прод: Redis, Mongo, SSE, безопасность). |
