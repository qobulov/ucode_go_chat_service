# Chat Service To'liq Dokumentatsiya

Bu hujjat [ucode_go_chat_service](https://github.com/Ucode-io/ucode_go_chat_service.git) loyihasining amaldagi kod bazasiga qarab yozildi. Maqsad faqat endpointlarni sanab chiqish emas, balki servis nima ish qiladi, qanday ishlaydi, qaysi qism nimaga javob beradi, qaysi field nimani anglatadi, qanday cheklovlar bor va qaysi joylarda noaniqlik yoki risk mavjudligini ham tushuntirish.

BaseURL: https://chat-service.u-code.io

## 1. Servis nima vazifa bajaradi

Bu servis real-time chat backend:

- chat room yaratadi
- userlarni roomlarga bog'laydi
- message history saqlaydi
- yangi xabarni socket orqali broadcast qiladi
- unread count hisoblaydi
- simple presence (`online`/`offline`) yuritadi
- typing indicator (yozayotganlik bildiruvchisi) qo'llab-quvvatlaydi
- REST va Socket.IO usullarini birga ishlatadi

Servis ichida alohida `users` jadvali yo'q. Servis user identifikatori sifatida tashqi tizimdan keladigan `row_id` bilan ishlaydi. Demak, bu backend user management tizimi emas, balki chat orchestration qatlamidir.

## 2. Ishlatilgan texnologiyalar

- `Go 1.24`
- `Gin` - HTTP router va REST API uchun
- `pgx/v5` - PostgreSQL connection pool uchun
- `Squirrel` - SQL query builder uchun
- `zerolog` - log yozish uchun
- `joho/godotenv` - `.env` yuklash uchun
- custom `internal/socketio` - Socket.IO server implementatsiyasi
- `PostgreSQL` - asosiy saqlash qatlami

## 3. Loyiha arxitekturasi

Yuqori darajadagi oqim:

1. `cmd/main.go` configni yuklaydi.
2. Logger yaratiladi.
3. PostgreSQL pool ochiladi.
4. Storage layer yig'iladi.
5. Gin router yaratiladi.
6. REST endpointlar register qilinadi.
7. Socket.IO endpoint ishga tushadi.
8. Presence sweeper fon goroutine sifatida start bo'ladi.

Asosiy qatlamlar:

- `cmd/` - entrypoint
- `config/` - env va konstanta
- `api/` - router va handlerlar
- `storage/` - repository abstraction
- `storage/postgres/` - SQL implementatsiya
- `models/` - request/response va domain model structlar
- `internal/socketio/` - custom socket server
- `migrations/` - schema
- `public/` - test UI

## 4. Ishga tushish oqimi

`cmd/main.go` quyidagini qiladi:

- `config.Load()` orqali envlarni oladi
- `logger.New(cfg.LogLevel)` bilan logger yasaydi
- `db.New(...)` orqali Postgres ulaydi
- `storage.New(...)` bilan repo layer yig'adi
- `api.New(...)` bilan Gin engine yasaydi
- `engine.Run(cfg.HTTPPort)` bilan serverni ishga tushiradi

Muhim envlar:

- `LOG_LEVEL`
- `HTTP_PORT`
- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DATABASE`
- `POSTGRES_MAX_CONNECTIONS`

Amalda `PostgresMaxConnections` configga o'qiladi, lekin `pgxpool` configga qo'llanmaydi. Ya'ni field bor, lekin hozirgi kodda ishlatilmayapti.

## 5. Router va tashqi interfeyslar

Servisda ikki xil interfeys bor:

- REST API
- Socket.IO real-time API

HTTP route'lar:

- `GET /` - test HTML sahifa
- `GET /static/*` - public fayllar
- `POST /v1/room`
- `GET /v1/room`
- `POST /v1/room/exist`
- `GET /v1/room/:item_id`
- `POST /v1/room-member`
- `GET /v1/message`
- `ANY /socket.io/*any`

### 5.1 REST contract qisqacha

#### `POST /v1/room`

Vazifasi:

- yangi room yaratish
- keyin creatorni member qilish
- agar `to_row_id` va `from_name` bo'lsa, ikkinchi memberni ham qo'shish

Asosiy request fieldlar:

- `name`
- `type`
- `project_id`
- `row_id`
- `to_name`
- `to_row_id`
- `from_name`
- `item_id`
- `attributes`

Response:

- `201`
- `body` ichida yaratilgan room

#### `GET /v1/room`

Query paramlar:

- `row_id` - majburiy
- `type` - optional
- `offset` - default `0`
- `limit` - default `10`, max `100`

Natija:

- current userga tegishli roomlar ro'yxati

#### `POST /v1/room/exist`

Vazifasi:

- room mavjudligini tekshirish

DM uchun:

- `project_id + type + row_id + to_row_id`

Group uchun:

- `project_id + type + item_id`

Natija:

- room id string yoki bo'sh string

#### `GET /v1/room/:item_id`

Query:

- `project_id` - majburiy

Natija:

- shu `item_id + project_id` uchun room id

#### `POST /v1/room-member`

Vazifasi:

- roomga member yozuvini qo'lda qo'shish

#### `GET /v1/message`

Query:

- `room_id` - majburiy
- `offset`
- `limit`

Natija:

- history

### 5.1.1 REST response formati

REST handlerlar bitta umumiy wrapper ishlatadi:

```json
{
  "body": {},
  "error": ""
}
```

Qoidalar:

- success bo'lsa `body` to'ladi, `error` bo'sh qoladi
- xato bo'lsa `error` to'ladi, `body` odatda `null` yoki bo'sh bo'ladi

Amaldagi xato statuslari:

- `400 Bad Request` - request noto'g'ri yoki majburiy field yo'q
- `500 Internal Server Error` - DB yoki ichki xato

Muhim:

- handlerlar gRPC style error matnlarini ham string sifatida qaytarishi mumkin
- xato response'lar structured error object emas, oddiy string

### 5.1.2 REST endpointlar bo'yicha to'liq misollar

#### `POST /v1/room` - room yaratish

Qachon ishlatiladi:

- yangi DM yaratish
- yangi group room yaratish

Minimal DM request misoli:

```bash
curl -X POST http://localhost:8080/v1/room \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "DM Room",
    "type": "single",
    "project_id": "592e6339-d867-489e-8e6a-74ea28e0818d",
    "row_id": "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e",
    "to_row_id": "515b0f7d-84c6-45ca-a600-5f9d1690a823",
    "to_name": "+998701121400",
    "from_name": "+998995002065"
  }'
```

Minimal group request misoli:

```bash
curl -X POST http://localhost:8080/v1/room \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Order Chat",
    "type": "group",
    "project_id": "592e6339-d867-489e-8e6a-74ea28e0818d",
    "row_id": "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e",
    "item_id": "2cebed90-9fc3-4e90-82e6-11727762dfc5",
    "to_name": "#order-123"
  }'
```

Success response odatda shunga o'xshaydi:

```json
{
  "body": {
    "id": "f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea",
    "name": "DM Room",
    "type": "single",
    "project_id": "592e6339-d867-489e-8e6a-74ea28e0818d",
    "item_id": "",
    "attributes": {},
    "created_at": "Mon, 17 Mar 2026 10:00:00 UTC",
    "updated_at": "Mon, 17 Mar 2026 10:00:00 UTC"
  },
  "error": ""
}
```

Xato holatlar:

- JSON noto'g'ri bo'lsa `400`
- `type` enumga mos kelmasa ko'pincha `500`, chunki DB reject qiladi
- `project_id`, `row_id` yoki UUID formatlar noto'g'ri bo'lsa DB xatosi bo'lishi mumkin

Noto'g'ri JSON misoli:

```bash
curl -X POST http://localhost:8080/v1/room \
  -H 'Content-Type: application/json' \
  -d '{"type":'
```

Xato response misoli:

```json
{
  "body": null,
  "error": "unexpected EOF"
}
```

#### `GET /v1/room` - user room listini olish

Qachon ishlatiladi:

- userga tegishli roomlar ro'yxatini olish
- pagination bilan list ko'rish
- `type=single` yoki `type=group` bilan filter qilish

Oddiy so'rov:

```bash
curl "http://localhost:8080/v1/room?row_id=48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e&offset=0&limit=20"
```

Faqat group roomlar:

```bash
curl "http://localhost:8080/v1/room?row_id=48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e&type=group&offset=0&limit=20"
```

Success response misoli:

```json
{
  "body": {
    "count": 2,
    "rooms": [
      {
        "id": "f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea",
        "name": "DM Room",
        "type": "single",
        "project_id": "592e6339-d867-489e-8e6a-74ea28e0818d",
        "to_name": "+998701121400",
        "to_row_id": "515b0f7d-84c6-45ca-a600-5f9d1690a823",
        "item_id": "",
        "attributes": {},
        "created_at": "Mon, 17 Mar 2026 10:00:00 UTC",
        "updated_at": "Mon, 17 Mar 2026 10:00:00 UTC",
        "last_message": "Salom",
        "last_message_type": "text",
        "last_message_file": "",
        "last_message_from": "+998995002065",
        "last_message_created_at": "Mon, 17 Mar 2026 10:05:00 UTC",
        "unread_message_count": 1,
        "user_presence_status": "online",
        "user_presence_last_seen": "Mon, 17 Mar 2026 10:05:10 UTC"
      }
    ]
  },
  "error": ""
}
```

Xato holatlar:

- `row_id` bo'lmasa `400`
- `limit` yoki `offset` son bo'lmasa `400`

`row_id` yo'q misoli:

```bash
curl "http://localhost:8080/v1/room?offset=0&limit=20"
```

Response:

```json
{
  "body": null,
  "error": "Row is required"
}
```

#### `POST /v1/room/exist` - room mavjudligini tekshirish

DM uchun misol:

```bash
curl -X POST http://localhost:8080/v1/room/exist \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "single",
    "project_id": "592e6339-d867-489e-8e6a-74ea28e0818d",
    "row_id": "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e",
    "to_row_id": "515b0f7d-84c6-45ca-a600-5f9d1690a823"
  }'
```

Group uchun misol:

```bash
curl -X POST http://localhost:8080/v1/room/exist \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "group",
    "project_id": "592e6339-d867-489e-8e6a-74ea28e0818d",
    "item_id": "2cebed90-9fc3-4e90-82e6-11727762dfc5"
  }'
```

Room topilsa:

```json
{
  "body": "f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea",
  "error": ""
}
```

Room topilmasa:

```json
{
  "body": "",
  "error": ""
}
```

Xato holatlar:

- JSON parse bo'lmasa `400`
- DB xato bo'lsa `500`

#### `GET /v1/room/:item_id` - item bo'yicha room id topish

Misol:

```bash
curl "http://localhost:8080/v1/room/2cebed90-9fc3-4e90-82e6-11727762dfc5?project_id=592e6339-d867-489e-8e6a-74ea28e0818d"
```

Success response:

```json
{
  "body": {
    "room_id": "f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea"
  },
  "error": ""
}
```

Topilmasa:

```json
{
  "body": {
    "room_id": ""
  },
  "error": ""
}
```

Xato holatlar:

- `project_id` bo'lmasa `400`
- `item_id` route param bo'sh bo'lsa `400`

Misol:

```bash
curl "http://localhost:8080/v1/room/2cebed90-9fc3-4e90-82e6-11727762dfc5"
```

Response:

```json
{
  "body": null,
  "error": "project_id is required"
}
```

#### `POST /v1/room-member` - roomga member qo'shish

Misol:

```bash
curl -X POST http://localhost:8080/v1/room-member \
  -H 'Content-Type: application/json' \
  -d '{
    "room_id": "f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea",
    "row_id": "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e",
    "to_name": "+998701121400",
    "to_row_id": "515b0f7d-84c6-45ca-a600-5f9d1690a823",
    "attributes": {}
  }'
```

Success response:

```json
{
  "body": {
    "id": "0f143d62-8ea3-43f3-8f61-ff9b5e902117",
    "room_id": "f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea",
    "row_id": "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e",
    "to_name": "+998701121400",
    "to_row_id": "515b0f7d-84c6-45ca-a600-5f9d1690a823",
    "attributes": {},
    "created_at": "Mon, 17 Mar 2026 10:10:00 UTC",
    "updated_at": "Mon, 17 Mar 2026 10:10:00 UTC"
  },
  "error": ""
}
```

Muhim:

- agar `(room_id, row_id)` oldin bor bo'lsa, insert `DO NOTHING` qiladi
- bunday holatda repository `nil, nil` qaytaradi
- REST response'da `body` `null` bo'lib qolishi mumkin

Bu xato emas, lekin contract nuqtai nazaridan noqulay holat.

#### `GET /v1/message` - room history olish

Misol:

```bash
curl "http://localhost:8080/v1/message?room_id=f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea&offset=0&limit=50"
```

Success response:

```json
{
  "body": {
    "count": 2,
    "messages": [
      {
        "id": "612f50d8-a789-4cc6-a819-f2449a0c3078",
        "room_id": "f3f95d20-5fc4-4cf9-a8df-65de2d0cb8ea",
        "message": "Salom",
        "type": "text",
        "file": "",
        "author_row_id": "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e",
        "from": "+998995002065",
        "parent_id": "",
        "created_at": "Mon, 17 Mar 2026 10:05:00 UTC",
        "updated_at": "Mon, 17 Mar 2026 10:05:00 UTC",
        "read_at": ""
      }
    ]
  },
  "error": ""
}
```

Xato holatlar:

- `room_id` bo'lmasa `400`
- `limit` yoki `offset` noto'g'ri bo'lsa `400`

Misol:

```bash
curl "http://localhost:8080/v1/message?limit=abc"
```

Response:

```json
{
  "body": null,
  "error": "strconv.Atoi: parsing \"abc\": invalid syntax"
}
```

### 5.2 Socket event contract qisqacha

Client -> server eventlar:

- `connected`
- `create room`
- `join room`
- `rooms list`
- `room history`
- `chat message`
- `presence:connected`
- `presence:ping`
- `presence:get`
- `message:read`
- `message:update`
- `typing:start`
- `typing:stop`
- `disconnected`

Server -> client eventlar:

- `rooms list`
- `room history`
- `chat message`
- `check room`
- `presence.updated`
- `message.read`
- `message.update`
- `typing:start`
- `typing:stop`
- `error`

Default limit qoidalari:

- socket handlerlar ko'p joyda `limit == 0` yoki `limit > 100` bo'lsa `100` qiladi

## 6. REST API nima uchun kerak

REST qismi asosan:

- room yaratish
- room mavjudligini tekshirish
- room list olish
- item bo'yicha room id topish
- room member qo'shish
- message history olish

Socket qismi esa:

- real-time ulanish
- room join
- message yuborish
- read status
- presence
- history push

Amalda frontendlar odatda Socket bilan ishlaydi, REST esa fallback yoki init uchun foydali.

## 7. Domain modeli

Servisdagi asosiy tushunchalar:

- `project_id`
- `row_id`
- `room`
- `room_member`
- `message`
- `presence`

### 7.1 `row_id` nima

`row_id` - bu chat servis ichidagi user primary key emas, balki tashqi tizimdan keladigan user identifikator.

Xulosa:

- userlar bu serviste yaratilmaydi
- `row_id` tashqi servisdan beriladi
- `row_id` format jihatdan UUID deb qabul qilinadi

### 7.2 `project_id` nima

`project_id` - tenant, loyiha yoki izolyatsiya domeni sifatida ishlatiladi.

Kodga qarab:

- `rooms` jadvalida `project_id` bor
- `messages`, `room_members`, `user_presence` jadvallarida `project_id` yo'q
- DM room qidirishda `project_id` hisobga olinadi
- group room qidirishda ham `project_id` hisobga olinadi
- `RoomIdByItemId` qidiruvida ham `project_id` ishlatiladi

Demak, `project_id` room scope uchun ishlatiladi, user scope uchun emas.

### 7.3 Har bir `project_id` uchun user unique bo'ladimi

Yo'q. Hozirgi schema va kod bo'yicha:

- `users` jadvali yo'q
- `row_id` uchun `project_id` bo'yicha unique constraint yo'q
- bitta `row_id` bir nechta `project_id`dagi roomlarda qatnashishi mumkin
- `user_presence` global bo'lib, faqat `row_id` bo'yicha yuritiladi

Demak:

- servis "har bir project uchun alohida user identity" modelini enforce qilmaydi
- hozirgi implementatsiyada user global identifikator sifatida qaraladi

## 8. Database schema to'liq tahlil

### 8.1 `rooms` jadvali

Fieldlar:

- `id` - room UUID, primary key
- `name` - room nomi
- `type` - `single` yoki `group`
- `item_id` - tashqi entity bilan bog'lash uchun optional identifikator
- `project_id` - room qaysi projectga tegishli ekanini bildiradi
- `created_at` - room yaratilgan vaqt
- `updated_at` - room yangilangan vaqt
- `attributes` - JSONB metadata

Semantika:

- `single` odatda ikki user orasidagi dialog
- `group` odatda item yoki umumiy kanalga bog'langan chat
- `item_id` ko'proq group chatni tashqi obyektga bog'lash uchun ishlatiladi

Muhim holat:

- `rooms` jadvalida `project_id + item_id + type` uchun unique index yo'q
- `single` roomlar uchun ham DB darajasida unique constraint yo'q
- duplicate roomlarning oldini olish faqat kod ichidagi `RoomExists` bilan qilinadi

Demak, parallel so'rovlar bo'lsa duplicate room paydo bo'lish riski bor.

### 8.2 `messages` jadvali

Fieldlar:

- `id` - message UUID
- `room_id` - qaysi roomga tegishli
- `message` - text content
- `type` - `text`, `image`, `video`, `voice`, `file`
- `file` - file URL yoki identifier
- `from` - yuboruvchi display qiymati
- `author_row_id` - haqiqiy yuboruvchi user identifikatori
- `read_at` - message o'qilgan vaqt
- `created_at` - yaratilgan vaqt
- `updated_at` - tahrirlangan vaqt
- `parent_id` - reply/thread uchun parent message

Muhim nuqta:

- `from` va `author_row_id` ikki xil maqsadga xizmat qiladi
- `from` odatda UI uchun name yoki telefon
- `author_row_id` esa identifikatsiya uchun

Muhim cheklov:

- `read_at` global message field
- bu field "har bir user uchun alohida read state" emas

Natija:

- group chatda bitta user o'qisa, message `read_at` to'lib qoladi
- bu boshqa userlar ham o'qigandek ko'rinish berishi mumkin
- shuning uchun `read_at` hozirgi dizaynda ko'proq "someone read it" ga yaqin, "everyone's per-user read status" emas

### 8.3 `room_members` jadvali

Fieldlar:

- `id` - row UUID
- `room_id` - qaysi room
- `row_id` - qaysi user
- `to_name` - UI helper field, ko'pincha qarshi tomon nomi
- `to_row_id` - qarshi tomon user idsi yoki target user
- `last_read_at` - aynan shu user uchun room bo'yicha oxirgi o'qilgan vaqt
- `created_at`
- `updated_at`
- `attributes` - memberga tegishli JSONB metadata

Unique constraint:

- `(room_id, row_id)` unique

Ma'nosi:

- bir user bitta roomga ikki marta member bo'lib qo'shilmaydi

Muhim:

- unread count aynan `last_read_at` orqali hisoblanadi
- bu per-user unread uchun to'g'ri asos
- shu sabab `room_members.last_read_at` juda muhim

### 8.4 `user_presence` jadvali

Fieldlar:

- `row_id` - user id, primary key
- `status` - `online` yoki `offline`
- `last_seen_at` - oxirgi heartbeat/aktivlik
- `created_at`
- `updated_at`

Muhim cheklov (va so'nggi yangilanish):

- Oldin jadval ichida xato bo'lib `project_id` qayd qilingan edi va DB da conflict berardi, bu muammo yaqinda to'liq yechildi.
- Endi jadval faqatgina `row_id` ni o'zini butunlay yagona (Primary Key) sifatida ushlab xavfsiz ishlaydi.
- presence global, project bo'yicha ajratilmagan
- bitta `row_id` uchun bitta satr bor

Natija:

- user bir projectda online bo'lsa, boshqa projectlar uchun ham online ko'rinishi mumkin
- bu multi-tenant isolation nuqtai nazaridan cheklov

## 9. Tablelar orasidagi bog'lanish

- `rooms.id -> messages.room_id`
- `rooms.id -> room_members.room_id`
- `messages.parent_id -> messages.id`

Lekin:

- `row_id` hech qaysi `users` jadvaliga foreign key emas
- `project_id` ham alohida `projects` jadvaliga foreign key emas

Demak, servis tashqi tizimlarga ishonadi va referential integrity'ni to'liq o'zi ta'minlamaydi.

## 10. Room turlari va ularning logikasi

### 10.1 `single` room

DM yaratishda server:

1. `project_id`, `type`, ikkala user kombinatsiyasi bo'yicha room bormi tekshiradi
2. bo'lsa yangi room ochmaydi, mavjud room idni qaytaradi
3. bo'lmasa room yaratadi
4. creator uchun `room_members` yozadi
5. target user uchun ham `room_members` yozadi

DM room mavjudligini tekshirish mezoni:

- `r.project_id = ?`
- `r.type = 'single'`
- room ichida `rm1.row_id = creator`
- room ichida `rm2.row_id = target`

Bu faqat kod qoidasi. DB unique constraint yo'q.

### 10.2 `group` room

Group room mavjudligi:

- `project_id`
- `type`
- `item_id`

Demak, group chat ko'pincha biror tashqi entityga bog'langan.

Masalan:

- ticket chat
- order chat
- object discussion
- channel-like room

## 11. Socket eventlar to'liq logikasi

### 11.1 `connected`

Vazifasi:

- userning socketi ulanishi bilan identifikatsiya qilish
- socketni `row_id` nomli roomga join qilish
- userni `online` qilish
- room list qaytarish

`row_id` roomga join qilishdan maqsad:

- keyinchalik `s.io.To(m.RowId).Emit(...)` qilib userga individual update push qilish

Bu yaxshi pattern:

- har user uchun private push channel

### 11.2 `create room`

Vazifasi:

- room yaratish yoki mavjud roomni topish
- member qo'shish
- clientga room list refresh qaytarish

Qo'llab-quvvatlaydigan qo'shimcha metadata:

- `attributes`
- `member_attributes`
- `to_member_attributes`

Bu metadata `JSONB` sifatida saqlanadi.

### 11.3 `join room`

Vazifasi:

- userni room socket channeliga join qilish
- `last_read_at` ni yangilash
- room history qaytarish
- yangi room list qaytarish

Amalda:

- user bir roomga join qilganda unread count kamayishi shu yerda boshlanadi

### 11.4 `rooms list`

Vazifasi:

- current userga tegishli roomlar ro'yxatini qaytarish
- **YANGILANISH:** Payload endi `project_id` ni qabul qiladi. DB hozir ushbu `project_id` ni tekshirib, foydalanuvchining alohida faqat o'sha projectdagi xonalarinigina filtrlab qaytaradi. Turli loyihalar aro xonalar endi aralashib ketmaydi.

Payloaddagi `type` bo'sh bo'lsa:

- ham `single`, ham `group` qaytadi

### 11.5 `room history`

Vazifasi:

- bitta roomning pagination bilan message historysini qaytarish

Server:

- DBda DESC oladi
- keyin clientga ASC qaytaradi

Maqsad:

- UIda eski -> yangi tartibda ko'rsatish

### 11.6 `chat message`

Vazifasi:

- message yaratish
- room ichiga broadcast qilish
- room memberlarining room listini yangilash

Oqim:

1. author uchun `last_read_at` yangilanadi
2. `messages` ga insert bo'ladi
3. `chat message` roomga emit qilinadi
4. room memberlari topiladi
5. har member uchun `rooms list` qayta hisoblanib push qilinadi

### 11.7 `message:read`

Vazifasi:

- room bo'yicha boshqa user yozgan unread xabarlarni read qilish
- `last_read_at` ni update qilish
- room ichiga `message.read` event yuborish

Muhim Yangilanish:

- payload `room_id` va `row_id` kutadi.
- Yaqinda qilingan tahrir tufayli, Socket va test UI contracti to'liq bog'landi: Tizim mutlaqo birgalashgan holda aniq `room_id` ni uzatadi va eski non-standart `message_id` endi yuborilmaydi.
- Bu tufayli avtomatik xonadagi Message Read statusi uzluksiz ishlaydi.

### 11.8 `message:update`

Vazifasi:

- `message`, `file`, `type` ni update qilish

Muhim cheklov:

- update event faqat o'sha socketga `message.update` qiladi
- room ichidagi boshqa userlarga broadcast qilmaydi

Demak, message edit multi-client sync hozir to'liq emas.

### 11.9 Typing indicator eventlar

Typing indicator — foydalanuvchi xabar yozayotganini boshqa room a'zolariga real-time ko'rsatish mexanizmi.

#### Eventlar:

- `typing:start` — foydalanuvchi yozishni boshladi
- `typing:stop` — foydalanuvchi yozishni to'xtatdi

#### Client -> Server payload:

```json
{
  "room_id": "c239d653-b50c-422b-bb49-77d16103a8bb",
  "user_name": "Shoh",
  "row_id": "369e2ebd-f486-41b2-99cf-efb803b827f5",
  "project_id": "b99fe3e9-6efb-4f75-b306-6b48f3d18ae0"
}
```

Fieldlar:

- `room_id` — majburiy, qaysi xonaga tegishli ekanini bildiradi
- `user_name` — UI da ko'rsatish uchun foydalanuvchi nomi
- `row_id` — presence yangilash uchun foydalanuvchi identifikatori
- `project_id` — presence heartbeat uchun loyiha identifikatori

#### Backend logikasi (`onTypingStart`):

1. Payloaddan `room_id` olinadi, bo'sh bo'lsa ignore qilinadi
2. Agar `row_id` va `project_id` mavjud bo'lsa, `PresenceHeartbeat` chaqiriladi — bu foydalanuvchining bazadagi `user_presence` jadvalidagi `last_seen_at` vaqtini yangilaydi va uni `online` holatida saqlab turadi
3. Payloadni aynan shu `room_id` ga ulangan barcha boshqa socketlarga broadcast qiladi (`s.io.To(roomId).Emit("typing:start", reqMap)`)

#### Backend logikasi (`onTypingStop`):

1. Payloaddan `room_id` olinadi
2. Payloadni room ichidagi barcha socketlarga broadcast qiladi
3. Presence yangilanmaydi (faqat `typing:start` da yangilanadi)

#### Frontend (client) tomoni qanday ishlaydi:

Debounce pattern ishlatiladi:

1. Foydalanuvchi xabar inputiga yozishni boshlasa, `typing:start` emit qilinadi
2. Har bir keypress da taymer qayta tiklanadi (1.5 soniya)
3. Agar 1.5 soniya davomida yangi keypress bo'lmasa, `typing:stop` emit qilinadi
4. Xabar yuborilganda ham `typing:stop` darhol emit qilinadi

```
User yoza boshlaydi -> typing:start emit
  |-- har keypress: taymer reset (1.5s)
  |-- 1.5s o'tdi, keypress yo'q -> typing:stop emit
  |-- yoki "Send" bosildi -> typing:stop emit
```

#### Listening (tinglash) tomoni:

Boshqa client `typing:start` eventini qabul qilganda:

1. Kim yozayotganini `user_name` dan oladi
2. UI da "Shoh is typing..." ko'rinishida animatsiya ko'rsatadi
3. `typing:stop` kelganda yoki xabar kelganda indikatorni yashiradi

#### Muhim xususiyatlar:

- Typing ma'lumotlari **bazaga saqlanmaydi** — faqat real-time broadcast
- `typing:start` har safar presence heartbeat ham bajaradi, shuning uchun tez-tez yozayotgan user hech qachon "offline" ga tushib qolmaydi
- Server o'zi typing holatini saqlamaydi, faqat clientlar orasida relay (uzatish) qiladi
- Bitta roomda bir nechta user bir vaqtda yozayotgan bo'lishi mumkin

#### Misol oqim:

```
Shoh yozishni boshlaydi:
  Client A -> Server: typing:start { room_id, user_name: "Shoh", row_id, project_id }
  Server -> Client B: typing:start { room_id, user_name: "Shoh", row_id, project_id }
  Server: PresenceHeartbeat(row_id, project_id) -> DB yangilanadi

Shoh 1.5s yozmaydi:
  Client A -> Server: typing:stop { room_id, user_name: "Shoh", ... }
  Server -> Client B: typing:stop { room_id, user_name: "Shoh", ... }

Shoh xabar yuboradi:
  Client A -> Server: typing:stop (darhol)
  Client A -> Server: chat message { ... }
  Server -> Room: chat message { ... }
```

### 11.10 Presence eventlar

Eventlar:

- `presence:connected`
- `presence:ping`
- `presence:get`
- `disconnected`

Logika:

- `connected` yoki `presence:connected` userni online qiladi
- `presence:ping` heartbeat bo'lishi kerak
- `presence:get` bitta user holatini qaytaradi
- `disconnected` offline qiladi

### 11.11 Presence sweeper

Fon goroutine har 1 daqiqada:

- `last_seen_at < now - 1 minute` bo'lgan userlarni topadi
- `offline` qiladi
- `presence.updated` event jo'natadi

Bu realtime presence fallback mexanizmi.

## 12. Room list qanday yig'iladi

`RoomGetList` query juda muhim. U room listni oddiy `rooms` jadvalidan bermaydi, balki quyidagilarni birlashtiradi:

- roomning o'zi
- `room_members` ichidagi `to_name`, `to_row_id`
- oxirgi message
- target user presence
- unread count

Clientga qaytadigan room object ichida shular bo'ladi:

- `last_message`
- `last_message_type`
- `last_message_file`
- `last_message_from`
- `last_message_created_at`
- `unread_message_count`
- `user_presence_status`
- `user_presence_last_seen`

Muhim dizayn nuqtasi:

- room list user-specific view
- bir room turli userlar uchun turli `to_name` va `to_row_id` bilan ko'rinishi mumkin

## 13. Unread count qanday hisoblanadi

Har room uchun:

- current user yozmagan message'lar olinadi
- `created_at > last_read_at` bo'lsa unread hisoblanadi

Bu to'g'ri per-user unread strategiya.

Lekin:

- `RoomGetList` ichida har room uchun alohida unread query ishlaydi
- roomlar soni ko'payganda bu `N+1 query` bo'ladi

## 14. Message ordering va pagination

History query:

1. DBdan `created_at DESC, id DESC` bilan oladi
2. `limit/offset` qo'llaydi
3. tashqarida ASCga qaytaradi

Bu pattern chat history uchun qulay, chunki:

- yangi xabarlar bo'yicha page olish oson
- UIga esa pastdan yuqoriga emas, vaqt tartibida qaytadi

## 15. `project_id` to'liq tahlil

`project_id` haqidagi asosiy xulosalar:

- bu field faqat `rooms` jadvalida mavjud
- `project_id` room tenantligini bildiradi
- `messages` va `room_members` `project_id`ni room orqali meros oladi
- `presence` esa roomdan mustaqil global

Bu nimani anglatadi:

- roomlar project bo'yicha ajraladi
- lekin user presence project bo'yicha ajralmaydi
- `row_id` global identifikator
- servis "har projectda alohida user namespace" modeliga ega emas

### 15.1 `project_id` bo'yicha unique user bormi

Yo'q.

Schema bo'yicha mavjud emas:

- `project_users` jadvali yo'q
- `user_presence(row_id, project_id)` yo'q
- `room_members(project_id, row_id)` yo'q

Shuning uchun bitta user:

- bir nechta projectdagi roomlarda qatnashishi mumkin
- presence esa bitta global state sifatida ko'rinadi

### 15.2 `project_id` bo'yicha unique room bormi

To'liq emas.

Kod darajasida:

- `single` room: project + 2 user kombinatsiyasi bo'yicha qidiriladi
- `group` room: project + item_id + type bo'yicha qidiriladi

Lekin DB darajasida:

- unique index yo'q

Demak, race condition holatida duplicate roomlar yuzaga kelishi mumkin.

## 16. `item_id` nima uchun kerak

`item_id` roomni tashqi obyektga bog'lash uchun ishlatiladi.

Misollar:

- task
- lead
- ticket
- order
- property

`GET /v1/room/:item_id?project_id=...` endpoint ham aynan shu bog'liqlik uchun ishlaydi.

## 17. `to_name` va `to_row_id` maydonlari nima

Bu fieldlar ayniqsa DM UI uchun kerak:

- `to_name` - qarshi tarafning UI nomi
- `to_row_id` - qarshi taraf identifikatori

Nega `room_members`da saqlanadi:

- bir room har member uchun turlicha ko'rinishi mumkin
- creator uchun `to_name = target name`
- target uchun `to_name = creator name`

Shuning uchun bu fieldlar room darajasida emas, member darajasida turibdi.

## 18. `attributes` maydonlari nima va qanday ishlatiladi

### 18.1 Umumiy tushuncha

`attributes` — bu `JSONB` formatdagi universal key-value maydon. Schema (jadval ustunlari) o'zgartirmasdan istalgan qo'shimcha ma'lumotni saqlash imkonini beradi.

Ikkita joyda `attributes` mavjud:

| Jadval | Maydon | Maqsad |
|---|---|---|
| `rooms` | `rooms.attributes` | Room darajasidagi umumiy metadata. **Room list javobida qaytadi.** |
| `room_members` | `room_members.attributes` | Har bir member uchun alohida metadata. Hozirgi kodda room list javobida **qaytmaydi.** |

Server attributes bilan quyidagicha ishlaydi:

- JSON map bo'lsa — marshal qilib saqlaydi
- string bo'lsa — valid JSON ekanini tekshirib saqlaydi
- hech narsa bo'lmasa — `{}` (bo'sh ob'yekt) saqlaydi

### 18.2 Profil rasm va meeting URL saqlash

Telemedicina, konsultatsiya yoki shunga o'xshash ilovalar uchun chat xonasining headerida **qarshi tarafning profil rasmi** va **meeting havolasi** ko'rsatish kerak bo'ladi.

Buning uchun `rooms.attributes` ichida ikkala foydalanuvchining ma'lumotlarini `row_id` kalit (key) sifatida saqlab qo'yish **eng yaxshi va oddiy yechim**:

#### Room yaratish payloadi:

```json
{
  "row_id": "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e",
  "project_id": "592e6339-d867-489e-8e6a-74ea28e0818d",
  "type": "single",
  "to_row_id": "515b0f7d-84c6-45ca-a600-5f9d1690a823",
  "to_name": "Dr. Kawsar",
  "from_name": "Ali",

  "attributes": {
    "meeting_url": "https://meet.google.com/xxx-xxxx-xxx",
    "meeting_date": "Онлайн запись 17 окт. 2025, 14:30",
    "profiles": {
      "48dc336f-17b6-4c0f-ad4b-d2e67b13ec2e": {
        "name": "Ali",
        "pic": "https://cdn.example.com/ali.jpg"
      },
      "515b0f7d-84c6-45ca-a600-5f9d1690a823": {
        "name": "Dr. Kawsar",
        "pic": "https://cdn.example.com/kawsar.jpg"
      }
    }
  }
}
```

#### Room list javobida qaytishi:

```json
{
  "id": "c239d653-b50c-422b-bb49-77d16103a8bb",
  "name": "...",
  "type": "single",
  "to_name": "Dr. Kawsar",
  "to_row_id": "515b0f7d-84c6-45ca-a600-5f9d1690a823",
  "attributes": {
    "meeting_url": "https://meet.google.com/xxx-xxxx-xxx",
    "meeting_date": "Онлайн запись 17 окт. 2025, 14:30",
    "profiles": {
      "48dc336f-...": { "name": "Ali", "pic": "https://cdn.example.com/ali.jpg" },
      "515b0f7d-...": { "name": "Dr. Kawsar", "pic": "https://cdn.example.com/kawsar.jpg" }
    }
  },
  "user_presence_status": "online",
  "user_presence_last_seen": "Mon, 24 Mar 2026 12:00:00 UTC"
}
```

### 18.3 Clientda qarshi tarafning profil rasmini olish

Room list javobida `to_row_id` maydoni **doim qarshi tarafning** IDsini ko'rsatadi:

- Ali kirganda: `to_row_id = "515b0f7d-..."` (ya'ni Dr. Kawsar)
- Dr. Kawsar kirganda: `to_row_id = "48dc336f-..."` (ya'ni Ali)

Shuning uchun client tomonda:

```dart
// Flutter / Dart misol
final room = roomsListItem;
final attrs = room['attributes'] ?? {};
final profiles = attrs['profiles'] as Map? ?? {};
final toRowId = room['to_row_id'];  // qarshi taraf IDsi

// Qarshi tarafning profil rasmi
final otherProfile = profiles[toRowId] ?? {};
final profilePic = otherProfile['pic'] ?? '';
final meetingUrl = attrs['meeting_url'] ?? '';

// UI da ishlatish
CircleAvatar(backgroundImage: NetworkImage(profilePic));

if (meetingUrl.isNotEmpty)
  GestureDetector(
    onTap: () => launchUrl(Uri.parse(meetingUrl)),
    child: Text(meetingUrl, style: TextStyle(color: Colors.blue)),
  );
```

```javascript
// JavaScript misol
const room = roomsListItem;
const attrs = room.attributes || {};
const profiles = attrs.profiles || {};
const toRowId = room.to_row_id;

// Qarshi tarafning profil rasmi
const otherPic = profiles[toRowId]?.pic || '';
const meetingUrl = attrs.meeting_url || '';
```

### 18.4 Nima uchun `rooms.attributes` ishlatish afzal

| Variant | Afzalligi | Kamchiligi |
|---|---|---|
| `rooms.attributes` | Room list javobida **avtomatik qaytadi**, backend o'zgartirish kerak emas | Ikkala user uchun bir xil ma'lumot ko'rinadi — client o'zi ajratishi kerak |
| `room_members.attributes` | Har user uchun alohida saqlash mumkin | Hozirgi kodda room list javobida **qaytmaydi** — backendga o'zgartirish kerak |

**Tavsiya:** `rooms.attributes` ichida `profiles` ob'yektini `row_id` bo'yicha kalit qilib saqlang. Client o'zi `to_row_id` yordamida qarshi tarafning ma'lumotini oladi.

### 18.5 Qo'shimcha metadata misollari

`attributes` ga har qanday key-value qo'shishingiz mumkin:

```json
{
  "profiles": { ... },
  "meeting_url": "https://meet.google.com/xxx",
  "meeting_date": "17 окт. 2025, 14:30",
  "room_icon": "https://cdn.example.com/room_icon.png",
  "department": "Kardiologiya",
  "priority": "high",
  "tags": ["urgent", "vip"],
  "custom_color": "#10b981"
}
```

Servis bu ma'lumotlarni shunchaki saqlaydi va qaytaradi — ularni talqin qilish (interpret) va ko'rsatish to'liq client (mobil ilova / frontend) vazifasidir.

## 19. Presence qanday ishlaydi

Normal niyat bo'yicha:

1. user ulanadi
2. online bo'ladi
3. heartbeat yuboradi
4. heartbeat to'xtasa sweeper offline qiladi

Amaldagi muammo:

- `presence:ping` payload `project_id` talab qiladi
- `user_presence` jadvalida `project_id` field yo'q
- `PresenceHeartbeat` SQL `ON CONFLICT (row_id, project_id)` ishlatadi

Bu schema bilan mos emas.

Natija:

- heartbeat flow noto'g'ri
- bu qism bug hisoblanadi

## 20. Read receipt logikasi to'liq to'g'rimi

Qisman.

To'g'ri ishlaydigan qism:

- `room_members.last_read_at` per-user unread hisoblash uchun ishlaydi

Cheklangan qism:

- `messages.read_at` global
- group chat uchun per-user receipt emas

Agar mahsulot talabi "har user uchun kim qachon o'qidi" bo'lsa, alohida `message_reads` yoki shunga o'xshash jadval kerak bo'ladi.

## 21. Xabar yuborilganda room tartibi yangilanadimi

UIga yangi `rooms list` push qilinadi, lekin DB query `rooms.updated_at DESC` bo'yicha sort qiladi.

Problem:

- yangi message kelganda `rooms.updated_at` update qilinmaydi
- demak room list order real oxirgi faollikka to'liq mos kelmasligi mumkin

Bu amaldagi logikadagi muhim kamchilik.

## 22. Xavfsizlik va auth holati

Socket auth:

- `OnAuthentication` hozir `return true`
- ya'ni auth yo'q

REST auth:

- routerda auth middleware yo'q

CORS:

- `AllowOrigins: *`

Demak, servis hozir development yoki trusted internal foydalanish modeliga yaqin. Production uchun auth va authorization qatlami yetishmaydi.

## 23. Error handling

REST:

- javob formati `{ body, error }`

Socket:

- `error` event emit qilinadi
- payloadda `function` va `message` bor

DB errorlar:

- `sql.ErrNoRows` -> not found
- `unique_violation` -> already exists
- `foreign_key_violation` -> invalid argumentga yaqin qaytariladi

## 24. Test UI nima qiladi

`public/index.html` ichida oddiy test client bor:

- socket ulaydi
- statik userlar bilan DM yaratadi
- room join qiladi
- message yuboradi
- log ko'rsatadi

Bu production frontend emas, test harness.

Muhim:

- test UI bilan backend contract to'liq mos emas
- ayniqsa `message:read` va `presence:ping` joylarida farq bor

## 25. Bu servisda yo'q narsalar

Kodga qarab mavjud emas:

- user registration/login
- JWT/token auth
- role/permission tekshiruvi
- project membership validation
- file upload storage
- delivery status
- per-user message receipt table
- deleted message logikasi
- message search
- push notification
- room archive/mute/pin

## 26. Muhim amaliy qoidalar

Servisdan foydalanganda quyidagini to'g'ri tushunish kerak:

- `row_id` tashqi systemdan keladi
- `project_id` room scope uchun
- `presence` global
- duplicate roomlardan himoya kod darajasida, DB darajasida emas
- group read receipt to'liq per-user emas
- room list sort hozir message activity bilan to'liq sync emas

## 27. Tavsiya etilgan ideal contract

Agar bu servisni barqaror product darajasiga olib chiqish kerak bo'lsa, quyidagilar kerak:

1. socket va REST auth qo'shish
2. `project_id` bo'yicha authorization qo'shish
3. DM/group room uniqueness uchun DB unique indexlar qo'shish
4. `user_presence` dizaynini project-aware yoki aniq global qilib yakuniylashtirish
5. `message_reads` jadvali qo'shish
6. `rooms.updated_at` ni yangi message kelganda update qilish
7. frontend/backend event contractni bitta stabil schema qilib yozish

## 28. Qisqa xulosa

Bu loyiha ishlaydigan chat backend skeleti:

- room yaratish bor
- member management bor
- history bor
- realtime broadcast bor
- unread count bor
- simple presence bor

Lekin u hozircha "full-featured production chat platform" emas. Kod bazasi ko'proq:

- ichki servis
- MVP
- integratsiya backend
- tashqi core systemga tayangan chat modul

sifatida ko'rinadi.

## 29. Eng muhim savollarga qisqa javob

### `project_id` nimani bildiradi

Room qaysi projectga tegishli ekanini bildiradi. User identityni ajratmaydi.

### Har bir `project_id` uchun user unique bo'ladimi

Yo'q. Bunga oid schema yoki constraint yo'q.

### `row_id` nima

Tashqi tizimdan keladigan global user identifikator.

### Presence project bo'yichami

Yo'q, hozirgi jadval bo'yicha global.

### DM duplicate bo'ladimi

Kod oldini olishga harakat qiladi, lekin DB unique index yo'qligi sabab race holatda bo'lishi mumkin.

### Group chat qaysi mezon bilan topiladi

`project_id + type + item_id`.

### Read status per-usermi

`unread count` per-user, lekin `messages.read_at` global.

### Room list oxirgi message bo'yicha sort bo'ladimi

To'liq emas, chunki `rooms.updated_at` yangi message bilan update qilinmaydi.
