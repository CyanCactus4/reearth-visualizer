# Capability: Real-time collaboration (живая спецификация)

Сводка требований из [TASK.md](../../TASK.md). Детализация и статус фаз — [PLAN.md](../../PLAN.md).

## ADDED Requirements

### Requirement: FR-1 WebSocket infrastructure

Система MUST предоставлять WS для проекта, комнаты по `projectId`, JWT, опционально Redis pub/sub между инстансами.

### Requirement: FR-2 State synchronization

Синхронизация MUST покрывать сущности из ТЗ (слои, стили, виджеты, storytelling, property) через согласованный `apply` и серверную нормализацию; атомарность операций на стороне сервера.

### Requirement: FR-3 Presence

MUST: список активных пользователей, курсоры с цветом, **видимой монограммой (avatar chip)** и подписью, индикаторы typing / move, блокировки на редактируемых объектах.

### Requirement: FR-4 Locks and conflicts

Optimistic locks, TTL, read-only для чужих блокировок; при конфликте — диалог выбора стратегии (reload / сравнение снимков в MVP).

### Requirement: FR-5 History and undo

Журнал apply, undo/redo только своих операций, авторы в UI, админский restore по снимкам где настроено.

### Requirement: FR-6 Notifications and chat

Toasts по событиям, чат, @mentions, опционально webhook.
