# Todo App with PostgreSQL

![Go](https://img.shields.io/badge/Go-1.20+-blue)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-blue)
![Gin](https://img.shields.io/badge/Gin-1.9+-brightgreen)

Todo-приложение с REST API на Go(Gin) и PostgreSQL

Todo App — это серверное приложение для управления списками задач (**Task Manager API**) с поддержкой:

- Авторизации и регистрации пользователей (JWT)
- CRUD для списков задач и отдельных задач
- Установки дедлайнов задач
- Редактирования и удаления
- PostgreSQL в качестве базы данных
- Миграций БД через [golang-migrate](https://github.com/golang-migrate/migrate)
- Swagger для интерактивной документации API
- **WebSocket‑уведомлений о приближении дедлайнов** (с возможностью воспроизведения звука на клиенте)

---

## Функциональность

### Авторизация (`/auth`)
- `POST /auth/sign-up` — регистрация нового пользователя
- `POST /auth/sign-in` — вход и получение JWT‑токена

### Списки задач (`/api/lists`)
- `POST /api/lists` — создание списка
- `GET /api/lists` — получение всех списков
- `GET /api/lists/:id` — получение конкретного списка
- `PUT /api/lists/:id` — обновление списка
- `DELETE /api/lists/:id` — удаление списка

### Задачи (`/api/lists/:id/items` и `/api/items`)
- `POST /api/lists/:id/items` — добавление задачи в список
- `GET /api/lists/:id/items` — получение задач в списке
- `GET /api/items/:id` — информация о задаче
- `PUT /api/items/:id` — обновление задачи (включая дедлайн и статус выполнения)
- `DELETE /api/items/:id` — удаление задачи

---

## WebSocket уведомления о дедлайнах

В приложение встроен модуль уведомлений в реальном времени:

- Подключение через эндпоинт: `ws://localhost:8000/ws`
- Авторизация по JWT
- При наступлении критического времени до дедлайна (например, ≤ 1 часа) сервер автоматически шлёт событие в реальном времени
- Формат сообщения:
{
"task_id":
42, "title": "Сдать
отчет", "deadline": "2025-08-20
15:00:00Z", "type":
text
- На клиенте можно прослушивать эти события и проигрывать **звуковые уведомления** 🔊, чтобы ничего не пропустить  

---

## 🛠️ Технологии

- Go 1.24+
- Gin (HTTP‑фреймворк)
- JWT (аутентификация)
- PostgreSQL + sqlx
- golang-migrate (миграции БД)
- Swagger (документация)
- logrus (логирование)
- WebSocket (уведомления о дедлайнах)

---

## Установка и запуск

### 1. Подготовить PostgreSQL
Создать базу данных:
createdb postgres

или через psql:
CREATE DATABASE postgres;

### 2. Настроить `.env`
Создай в корне `.env` файл:
DB_PASSWORD=qwerty

### 3. Миграции
Выполнить:
migrate -path ./migrations -database "postgres://postgres:qwerty@localhost:5432/postgres?sslmode=disable" up

Таблицы:
- `users`
- `todo_lists`
- `users_lists`
- `todo_items`
- `lists_items`

### 4. Запуск сервера
go run main.go

### 5. Доступ
- API: `http://localhost:8000`
- Swagger UI: `http://localhost:8000/swagger/index.html`
- WebSocket: `ws://localhost:8000/ws`

