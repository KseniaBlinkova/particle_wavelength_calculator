# Лабораторная работа №3 — веб-сервис частиц

Backend для каталога частиц. Сервис хранит данные в PostgreSQL через ORM
GORM, изображения и короткие видео — в MinIO. Настоящей авторизации в этой
лабораторной нет: во всех операциях используется фиксированный пользователь
`id=1`, получаемый через функцию-singleton `auth.Get()`.

## Быстрый запуск

Требования: Docker Desktop и Go 1.27 или новее.

1. Запустить PostgreSQL, MinIO и Adminer:

   ```bash
   docker compose up -d
   docker compose ps
   ```

2. Запустить backend из корня проекта:

   ```bash
   go run ./cmd/particleserver
   ```

3. Проверить сервис:

   ```bash
   curl http://localhost:8081/api/health
   ```

   Ожидаемый ответ:

   ```json
   {"api":"/api","bucket":"de-broglie"}
   ```

После запуска доступны:

- API: `http://localhost:8081/api`;
- MinIO API и публичные файлы: `http://localhost:9010`;
- консоль MinIO: `http://localhost:9011` (`minioadmin` / `minioadmin`);
- Adminer: `http://localhost:8090`.

Проект является только REST backend: HTML-шаблонов и server-rendered страниц в
нём нет. Все пользовательские сценарии выполняются через методы `/api/...`.

Вход в Adminer: система `PostgreSQL`, сервер `postgres_particle`, пользователь
`particle_user`, пароль `particle_pass`, база `particle_db`. При подключении к
PostgreSQL не из Docker используйте `localhost:5433`.

Настройки по умолчанию находятся в `internal/config/config.go`. Их можно
переопределить переменными `HTTP_ADDR`, `DATABASE_DSN`, `MINIO_ENDPOINT`,
`MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL` и
`MINIO_PUBLIC_URL`.

По умолчанию backend использует MinIO из `docker-compose.yml` на портах
`9010/9011`. Если нужно подключить другой экземпляр MinIO, endpoint можно
переопределить; публичный URL вычисляется из него автоматически:

```bash
MINIO_ENDPOINT=localhost:9000 go run ./cmd/particleserver
```

При запуске сервер выполняет GORM `AutoMigrate`, поэтому таблицы, внешние ключи
и уникальные индексы создаются при подключении к пустой базе автоматически.
Пользователь `id=1` также гарантированно создаётся сервером.

> `docker compose down -v` удаляет контейнеры вместе со всеми данными PostgreSQL
> и MinIO. Используйте эту команду только когда действительно нужна чистая база.

## HTTP API

Успешные ответы API имеют формат JSON, кроме удаления и выхода с кодом
`204 No Content`. При ошибке сервис возвращает только стандартный HTTP-код
(`400`, `403`, `404`, `409`, `413`, `415` или `500`) с пустым телом. Код ответа
не дублируется внутри JSON.

### Домен «услуга/частица»

- `GET /api/particles` — возвращает список опубликованных услуг; фильтры `name` и `min_mass` необязательны.
- `GET /api/particles/feed` — возвращает первую опубликованную услугу ленты.
- `GET /api/particles/feed?id={id}` — возвращает опубликованную услугу по ID.
- `GET /api/particles/feed?id={id}&next=true` — возвращает следующую опубликованную услугу после указанного ID.
- `GET /api/particles/draft` — возвращает единственный черновик текущего пользователя; ID с клиента не передаётся.
- `POST /api/particles` — создаёт черновик; принимает `name`, `mass`, `spin`, `full_description` и файлами `image`, `video`.
- `PUT /api/particles/{id}/publish` — публикует свой черновик; в JSON можно передать `mass`, `spin`, `full_description`.
- `DELETE /api/particles/{id}` — логически удаляет свою услугу: ставит статус `deleted` и дату `completed_at`.
- `POST /api/particles/{id}/like` — ставит (`liked=1`) или снимает (`liked=0`) лайк текущего пользователя.

Для изображения разрешены JPEG, PNG и WebP до 10 МБ. Для видео разрешены MP4
и WebM до 50 МБ. Полный multipart-запрос ограничен 64 МБ. Имя файла
транслитерируется в латиницу, получает дату и случайный суффикс; в БД хранится
ключ объекта, а сам файл помещается в MinIO.

Если необязательная масса не передана или поле формы оставлено пустым, в БД
сохраняется `NULL`: значение `0` автоматически не подставляется. В JSON-ответе
поле `mass` присутствует всегда и для такой услуги равно `null`. При фильтрации по `min_mass`
записи без массы не включаются.

Список и лента упорядочены по `id` по возрастанию. Масса используется только
для фильтра `min_mass` и не влияет на порядок записей.

Внутренний статус услуги хранится в таблице `particles` и используется только
бизнес-логикой backend. В JSON услуги поле `status` не возвращается.

### Домен «пользователь»

- `POST /api/users` — регистрирует пользователя по `username`, `email`, `password`;
- `POST /api/users/login` — заглушка входа; принимает `email`, `password`
- `POST /api/users/logout` — заглушка выхода; возвращает `204 No Content`.


- `GET /api/health` — проверяет PostgreSQL и возвращает имя бакета MinIO.

Системные поля `id`, `status`, `creator_id`, `created_at`,
`formed_at` и `completed_at` не принимаются из запросов на изменение. Они
вычисляются бизнес-логикой backend. Удалённые записи не возвращаются списком и
не попадают в ленту.

Переход `published → draft` и любые переходы из `deleted` запрещены.

## Таблицы базы данных

### `users`

Хранит зарегистрированных пользователей.

- `id` — `SERIAL`, первичный ключ.
- `username` — `VARCHAR(50)`, имя пользователя.
- `email` — `VARCHAR(100)`, уникальная электронная почта.
- `password_hash` — `VARCHAR(255)`, bcrypt-хеш пароля.
- `created_at` — `TIMESTAMP`, дата регистрации.

### `particles`

Хранит черновики, опубликованные и логически удалённые услуги.

- `id` — `SERIAL`, первичный ключ.
- `name` — `VARCHAR(100)`, название.
- `full_description` — `TEXT`, полное описание.
- `status` — `VARCHAR(20)`: `draft`, `published` или `deleted`.
- `image_key` — `VARCHAR(255)`, имя изображения в MinIO.
- `video_key` — `VARCHAR(255)`, имя видео в MinIO.
- `mass` — `DOUBLE PRECISION`, необязательная масса частицы; допускает `NULL`.
- `spin` — `VARCHAR(10)`, спин.
- `wavelength` — `DOUBLE PRECISION`, длина волны предметной области.
- `creator_id` — `INTEGER`, внешний ключ на `users.id`.
- `created_at` — `TIMESTAMP`, дата создания.
- `formed_at` — `TIMESTAMP`, дата публикации; допускает `NULL`.
- `completed_at` — `TIMESTAMP`, дата логического удаления; допускает `NULL`.

### `likes`

Связывает пользователей и услуги, которым они поставили лайк.

- `id` — `SERIAL`, первичный ключ.
- `user_id` — `INTEGER`, внешний ключ на `users.id`.
- `particle_id` — `INTEGER`, внешний ключ на `particles.id`.
- `created_at` — `TIMESTAMP`, дата установки лайка.

Пара `(user_id, particle_id)` уникальна.
Наличие такой строки означает `is_liked = 1` для текущего пользователя;
отсутствие строки — `is_liked = 0`. Отдельное поле в таблице `particles` не
требуется, потому что признак зависит одновременно от услуги и пользователя.

## Структура проекта

```text
cmd/particleserver       точка запуска
internal/domain          HTTP-обработчики, DTO/сериализаторы и бизнес-логика
internal/models          модели GORM и статусы
internal/storage         PostgreSQL-репозитории
internal/filestorage     работа с MinIO
internal/auth            singleton текущего пользователя
internal/httpapi         общий router и middleware
resources/static/media   файлы-заглушки для первоначальной загрузки в MinIO
```

## Проверка кода

```bash
go vet ./...
```
