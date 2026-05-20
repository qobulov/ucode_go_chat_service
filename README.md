# ucode_go_chat_service

Real-time chat microservice (Go, PostgreSQL, Socket.IO).

## Documentation

- [Full service documentation (UZ)](docs/chat_service_full_documentation_uz.md)
- [Flutter integration guide (UZ)](docs/flutter_chat_integration_uz.md)

## REST API (`/v1`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/room` | Create room |
| GET | `/room` | List rooms for `row_id` |
| PATCH | `/room/:id` | Update room `name`, `attributes` |
| DELETE | `/room/:id` | Delete room |
| POST | `/room-member` | Add member |
| PATCH | `/room-member` | Update member `to_name`, `attributes`, `from_name` |
| GET | `/message` | Message history |
| GET | `/supervisor/rooms` | All rooms by `project_id` (read-only) |
| GET | `/supervisor/messages` | Messages by `room_id` (read-only) |

## Socket events (client -> server)

`connected`, `create room`, `join room`, `rooms list`, `chat message`, `room history`, `presence:*`, `message:read`, `message:update`, `typing:start`, `typing:stop`, `room:delete`, `room:leave`, `room:update`, `disconnected`

## Socket events (server -> client)

`rooms list`, `room history`, `chat message`, `check room`, `presence.updated`, `message.read`, `message.update`, `room.deleted`, `room.member_left`, `room.updated`, `typing:*`, `error`

## Local run

Configure PostgreSQL in `.env`, run migrations, then:

```bash
go run .
```

Test UI: `GET /` serves `public/index.html`.
