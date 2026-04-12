# OpenSpec — Re:Earth Visualizer (лабораторная 5)

Проект использует [OpenSpec](https://github.com/Fission-AI/OpenSpec) для SDD поверх brownfield-кода.

- Функциональное ТЗ и трассировка: [TASK.md](../TASK.md), [PLAN.md](../PLAN.md).
- Контракт протокола (фаза 0): [docs/design-doc/20260411_001_collaboration_protocol_mvp.md](../docs/design-doc/20260411_001_collaboration_protocol_mvp.md).

## Cursor

После `npx @fission-ai/openspec@latest init . --tools cursor` в репозитории доступны команды в `.cursor/commands/` (например `/opsx:explore`, `/opsx:propose`, `/opsx:apply`, `/opsx:archive`) и навыки в `.cursor/skills/`.

Рекомендуемый цикл из **TASK.md**: исследование → предложение изменения → реализация в коде → проверка (тесты) → архивация изменения.
