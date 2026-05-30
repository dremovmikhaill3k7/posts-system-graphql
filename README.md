# Posts Service

GraphQL-сервис постов и вложенных комментариев (Go, gqlgen).

- Playground: http://localhost:8080/
- API: http://localhost:8080/v1/query

## Запуск без базы

```bash
cd backend
export JWT_SECRET=dev-secret-change-me-min-32-chars
export STORAGE_MODE=memory
go run ./cmd/api/main.go
```

Данные в памяти — после перезапуска пропадают.

## Запуск с Postgres

В корне проекта скопируй и заполни `.env` (есть пример в репозитории). Нужны как минимум `POSTGRES_*`, `JWT_SECRET`, `STORAGE_MODE=postgres`.

```bash
docker compose up -d db
make up
docker compose up --build
```

Откат миграции: `make down`. Новая: `make create name=имя`.

Локально API + БД в Docker: в `.env` поставь `POSTGRES_HOST=localhost`, подними `db`, сделай `make up`, затем `go run` из `backend` с теми же переменными.

## Переменные

- `STORAGE_MODE` — `postgres` (по умолчанию) или `memory`
- `JWT_SECRET` — обязателен
- `JWT_TTL_HOURS` — по желанию, по умолчанию 168
- `COOKIE_SECURE` — `true` для HTTPS
- `POSTGRES_*` — для postgres-режима

## Авторизация

После `register` или `login` сервер ставит http-only cookie `auth_token`. Дальше мутации (`createPost`, `createComment`, `updatePost`) работают из того же браузера на localhost без заголовков. Выход — `logout`.

## Пагинация

Везде опциональные `limit` и `offset`. Если не передать — `limit=20`, `offset=0`, максимум `limit=100`.

Работает в `posts`, `users`, `User.posts`, `Post.comments`, `replyComments`.

Комментарии к посту — только корневые (`Post.comments`). Ответы на комментарий — `replyComments(commentId: ...)`, для следующего уровня снова `replyComments`, но уже с id дочернего комментария. Писать ответ — всегда `createComment` с `parent_id`. У комментария поле `has_replies` — есть ли прямые ответы (без загрузки списка).

Следующая страница: увеличь `offset` на `limit`. Если пришло меньше записей, чем `limit`, — данные кончились. Поля `totalCount` нет.

## Комментарии

Создание: `createComment`. Корень — без `parent_id`, ответ — с `parent_id`. Текст до 2000 символов. Автор поста может выключить комментарии: `updatePost` с `can_have_comm: false`.

Новые комментарии в реальном времени: subscription `commentAdded(postId)` (WebSocket в Playground).

## Тесты

```bash
cd backend && go test ./...
```

## gqlgen

```bash
cd backend && go run github.com/99designs/gqlgen generate
```

Схема: `backend/internal/graph/schema.graphqls`.
