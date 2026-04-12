# Команды для автозапуска (allowlist для агента)

Разрешите выполнение этих команд без ручного подтверждения, чтобы агент мог собирать проект, гонять тесты и линтеры. Все пути относительно корня репозитория `reearth-visualizer`, если не указано иное.

## Общие

- `git status`
- `git diff`
- `git log -n 20 --oneline`

## Git — коммиты и ветки (после каждой законченной функциональности)

Разрешите эти команды, чтобы агент мог фиксировать работу в истории без ручного ввода. Сообщения коммитов — **Conventional Commits** со scope, как в [server/AGENTS.md](server/AGENTS.md) (например `feat(server): …`, `fix(web): …`, `test(e2e): …`).

- `git add -A`
- `git add -p`
- `git add <path>` (любые пути в репозитории)
- `git restore <path>` / `git restore --staged <path>`
- `git reset` / `git reset --soft HEAD~1` (осторожно: только при явной задаче отката коммита)
- `git commit -m "<message>"`
- `git commit -m "<title>" -m "<body>"`
- `git branch`
- `git checkout -b <branch>` / `git switch -c <branch>`
- `git checkout <branch>` / `git switch <branch>`
- `git merge <branch>` (по запросу пользователя или в рамках согласованного workflow)
- `git stash push -m "<msg>"` / `git stash pop` / `git stash list`
- `git show --stat HEAD`

**Отправка на удалённый репозиторий** (включайте в allowlist только если доверяете агенту пушить в ваш remote):

- `git remote -v`
- `git push -u origin <branch>`
- `git push`

**Не включать без необходимости:** `git push --force`, `git clean -fdx`, переписывание истории на `main`.

## Сервер (Go) — каталог `server/`

- `cd server && make help` (справка по целям)
- `cd server && make build`
- `cd server && make test`
- `cd server && make test-debug`
- `cd server && go test ./... -short -count=1`
- `cd server && go test ./internal/... -count=1`
- `cd server && go test ./pkg/... -count=1`
- `cd server && go test -v ./e2e/...` (или `make e2e` — см. `server/Makefile`)
- `cd server && make lint` / `cd server && make d-lint`
- `cd server && make gql` / `cd server && make generate`
- `cd server && gofmt -s -w .`
- `cd server && go mod tidy`

## Сервер + Docker (интеграция, e2e)

- `docker compose ps`
- `docker compose up -d` / `docker compose down`
- `cd server && make d-run` (полный стенд из README)
- `cd server && make d-run-db`
- `cd server && make d-test`
- `cd server && make e2e` (см. `server/mk/local.mk`: `go test -v ./e2e/...`)

## Веб (React + Vite) — каталог `web/`

- `cd web && yarn install`
- `cd web && yarn start` (долгоживущий процесс — только при необходимости дебага)
- `cd web && yarn build`
- `cd web && yarn test`
- `cd web && yarn coverage`
- `cd web && yarn type`
- `cd web && yarn lint`
- `cd web && yarn fix`
- `cd web && yarn gql`

## E2E (Playwright) — каталог `e2e/`

- `cd e2e && npm ci` или `yarn install` (смотреть lockfile в каталоге)
- `cd e2e && npm run test:api:local` (локальный API-прогон с mock auth)
- `cd e2e && npm run test:local` (UI, нужен поднятый фронт)

## Поиск по коду (read-only)

- `rg`, `find`, `head`, `wc`

---

**Примечание.** Команды, которые поднимают сервер на портах или пишут в Mongo/GCS, требуют запущенного Docker или локальных сервисов. Деструктивные цели вроде `make d-reset-data` лучше оставлять с явным подтверждением человека.

**Коммиты.** После прохождения релевантных тестов/линтера для добавленной функциональности агент делает **отдельный коммит** с осмысленным сообщением (см. раздел «Git» выше и [AGENTS.md](AGENTS.md)).
