# Лабораторная работа №4: что показать и что говорить

## Что именно требовалось реализовать

1. Регистрацию с уникальным логином (пароль хранится без хеширования, по требованию преподавателя).
2. Аутентификацию: проверку логина и пароля и выдачу JWT.
3. Авторизацию: публичное чтение списка и ленты, защищённые методы создателя.
4. Автоматическое заполнение `creator_id` из JWT, а не из тела запроса.
5. Logout через Redis blacklist: отозванный JWT больше не принимается.
6. Swagger/OpenAPI с Bearer Auth и описанием всех методов.
7. Полный бизнес-процесс на sequence-диаграмме.

В Go по заданию используются JWT и blacklist. Cookie и серверные сессии нужны
для сравнения на контрольных вопросах, но не являются механизмом авторизации
этого проекта.

## Девять скриншотов

Перед защитой выполнить `docker compose up -d` и `go run ./cmd/particleserver`.

1. Открыть в режиме инкогнито `http://localhost:8081/swagger/index.html`,
   выполнить `POST /users/login`, показать `200` и поле `token`.
2. Нажать **Authorize**, вставить JWT без слова `Bearer`, показать активную
   авторизацию.
3. В Swagger выполнить `GET /particles`, показать `200` и список.
4. В Postman/Insomnia удалить заголовок Authorization и выполнить
   `GET /api/particles`: должен быть `200`.
5. Добавить `Authorization: Bearer <JWT>` и повторить тот же GET: снова `200`.
6. Без Authorization выполнить `PUT /api/particles/{draft_id}/publish`:
   должен быть `401 Unauthorized`.
7. С JWT владельца повторить PUT с JSON, например
   `{"mass":2.5,"spin":"1","full_description":"updated"}`. Показать `200`
   и обновлённые поля. До этого нужно создать черновик через `POST /particles`.
8. Выполнить `POST /users/logout`, затем в терминале показать Redis:

   ```bash
   docker exec -it particle_redis redis-cli
   KEYS jwt:blacklist:*
   GET jwt:blacklist:<jti>
   TTL jwt:blacklist:<jti>
   ```

   Значение содержит `user_id`. Повторный защищённый запрос со старым JWT
   должен вернуть `401`.
9. В StarUML открыть sequence-диаграмму и показать, что у Message заполнено
   свойство `signature`, выбранное из операции соответствующего Domain на
   диаграмме классов.

## Техническое задание

### Назначение

REST backend должен поддерживать публичный просмотр опубликованных услуг и
защищённый бизнес-процесс их создания, публикации, удаления и оценки.

### Роли

- Гость: регистрация, вход, health, список, фильтр и лента.
- Авторизованный пользователь: возможности гостя, свой черновик, создание,
  публикация и удаление своих услуг, лайк, logout.

### Правила доступа

- Нет JWT на защищённом маршруте: `401`.
- JWT неверен, просрочен или находится в blacklist: `401`.
- JWT корректен, но услуга принадлежит другому пользователю: `403`.
- `creator_id` никогда не принимается от клиента.
- Пароль не возвращается клиенту в ответах API.

### Хранилища

- PostgreSQL: пользователи, услуги и лайки.
- MinIO: изображения и видео.
- Redis: `jwt:blacklist:<jti>` со значением `user_id=<id>` и TTL до `exp` JWT.

## Sequence-диаграмма

Не добавлять PostgreSQL и нативное приложение. Рекомендуемые Lifeline:

- `User` — Actor;
- `Swagger / SPA` — Boundary;
- `AuthMiddleware` — Control;
- `UserHandler` и `ParticleHandler` — Control;
- `UserDomain` и `ParticleDomain` — домены;
- `FileStorage` — только для загрузки image/video, если преподаватель требует
  показать внешний сервис.

Сначала добавить операции на диаграмму классов, затем выбирать их в поле
`signature` сообщений StarUML:

```text
UserDomain.Register(request: RegisterRequest): UserResponse
UserDomain.Login(request: LoginRequest): LoginResponse
UserDomain.Logout(): void
ParticleDomain.List(filter: Filter): ListResponse
ParticleDomain.Create(request: CreateRequest): ParticleResponse
ParticleDomain.Draft(): ParticleResponse
ParticleDomain.Publish(id: uint, request: PublishRequest): ParticleResponse
ParticleDomain.Feed(id: uint?, next: bool): FeedResponse
ParticleDomain.Like(id: uint, liked: int): LikeResponse
```

Порядок сообщений бизнес-процесса:

1. `POST /users` → `Register(login, password)` → пользователь.
2. `POST /users/login` → `Login(login, password)` → JWT и пользователь.
3. `GET /particles?min_mass=` → `List(filter)` → список.
4. `POST /particles` с JWT и multipart → middleware проверяет JWT →
   `Create(name, mass, spin, description, image, video)` → черновик.
5. `GET /particles/draft` с JWT → `Draft()` → свой черновик.
6. `PUT /particles/{id}/publish` с JWT → проверка владельца →
   `Publish(id, mass, spin, description)` → опубликованная услуга.
7. `GET /particles/feed` → `Feed(nil, false)` → первый элемент.
8. `GET /particles/feed?id={id}&next=true` → `Feed(id, true)` → следующий.
9. `POST /particles/{id}/like` с JWT → `Like(id, 1)` → состояние лайка.

Возврат каждого сообщения обозначить пунктирной стрелкой и подписать ключевые
выходные данные: HTTP status, `token`, `user`, `items`, `particle`, `likes_count`.

## Короткие ответы на контрольные вопросы

### Cookie

Небольшая пара имя/значение, которую браузер хранит и автоматически отправляет
серверу для подходящих домена и пути. Флаги `HttpOnly`, `Secure` и `SameSite`
снижают риск кражи и CSRF.

### Сессия

Состояние входа хранится на сервере, обычно в Redis. В cookie находится только
случайный session ID. Сессию легко немедленно удалить, но серверу нужно общее
хранилище состояния.

### Redis

Быстрое in-memory key-value хранилище с TTL. Подходит для сессий, кэша,
rate limiting и blacklist JWT.

### JWT

Подписанный токен из header, payload и signature. Payload читается, но не может
быть незаметно изменён без нарушения подписи. `exp` задаёт срок действия, `sub`
идентифицирует пользователя, `jti` идентифицирует конкретный токен.

### Аутентификация и авторизация

Аутентификация отвечает «кто это?» — проверяет пароль и выдаёт JWT.
Авторизация отвечает «что ему разрешено?» — проверяет наличие JWT и владение
услугой.

### SSO

Single Sign-On позволяет один раз войти у общего Identity Provider и затем
использовать несколько связанных приложений без повторного ввода пароля.

### Двухфакторная аутентификация

Требует два разных фактора: знание (пароль), владение (телефон/ключ) или
биометрию. Два пароля не считаются двумя факторами одного типа.

### RSA

Асимметричный алгоритм: приватный ключ подписывает, публичный проверяет.
JWT с RS256 удобен, когда токен выпускает один сервис, а проверяют несколько.
В этой лабораторной JWT подписывается HS256 общим секретом; RSA нужно понимать
как альтернативный способ подписи JWT.
