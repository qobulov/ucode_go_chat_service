# Flutter Chat Integratsiya Qo'llanmasi (DM + Group)

Bu hujjat Flutter ilovasida `ucode_go_chat_service` bilan to'liq chat ishlatish uchun yozilgan.
Maqsad: ketma-ket, amaliy, productionga yaqin yo'l xaritasi berish.

## 1. Scope

Qo'llanma quyidagilarni qamrab oladi:

- Socket ulanish va lifecycle
- Room list
- DM yaratish
- Group yaratish
- Roomga kirish va history
- Xabar yuborish
- Read receipt
- Typing indicator
- Presence
- Reconnect va basic reliability

## 2. Tavsiya etilgan package'lar

Minimal tavsiya:

- `socket_io_client`
- `dio` (REST uchun)
- `flutter_riverpod` yoki `bloc` (state management)
- `freezed` + `json_serializable` (model)
- `hive` yoki `isar` (offline cache, optional)

## 3. Tavsiya etilgan arxitektura

`lib/` ichida:

- `data/chat_socket_service.dart`
- `data/chat_rest_service.dart`
- `domain/chat_repository.dart`
- `domain/models/...`
- `application/chat_controller.dart`
- `presentation/rooms_page.dart`
- `presentation/chat_page.dart`

Masuliyat bo'linishi:

- `ChatSocketService`: socket connect/emit/listen
- `ChatRestService`: REST endpointlar
- `ChatRepository`: business flow (DM create, join, send, read)
- `ChatController`: UI state va actionlar

## 4. Event mapping (Flutterda listenerlar)

Serverdan tinglanadigan eventlar:

- `rooms list`
- `room history`
- `chat message`
- `check room`
- `message.read`
- `presence.updated`
- `typing:start`
- `typing:stop`
- `error`

Clientdan yuboriladigan asosiy eventlar:

- `connected`
- `create room`
- `join room`
- `rooms list`
- `chat message`
- `message:read`
- `typing:start`
- `typing:stop`
- `presence:ping`
- `disconnected`

## 5. Boshlang'ich ketma-ketlik (App start)

1. User login bo'lganidan keyin `row_id` va `project_id`ni oling.
2. Socket connect qiling (`websocket transport`).
3. `onConnect` ichida darhol `connected` event yuboring.
4. 30 sekund interval bilan `presence:ping` timer boshlang.
5. `rooms list` event kelganda roomlarni statega yozing.
6. App backgroundga o'tsa typing va timerlarni tozalang.

`connected` payload:

```json
{
  "row_id": "user-uuid",
  "project_id": "project-uuid",
  "offset": 0,
  "limit": 50
}
```

## 6. DM flow (to'liq, duplicate-safe)

1. Foydalanuvchi peer tanlaydi (`to_row_id`, `to_name`).
2. Har doim bir xil login user uchun bir xil `row_id` ishlatilishini tekshiring.
3. `create room` (`type: single`) ni Socket orqali yuboring.
4. Server javobi:
- Mavjud room bo'lsa `check room` (`room_id`) qaytadi.
- Yangi room ochilgan bo'lsa creatorga `rooms list` push qilinadi.
5. `check room` kelganda darhol `join room` yuboring.
6. `room history` kelgach chat ekranini to'ldiring.
7. Xabar yuborishda `chat message` ishlating.

Muhim:

- DM ochishda `POST /v1/room` ni to'g'ridan-to'g'ri ishlatmang.
- `from_name` ni bo'sh yubormang.

`create room` (DM) payload:

```json
{
  "row_id": "my-row-id",
  "project_id": "project-id",
  "type": "single",
  "to_row_id": "peer-row-id",
  "to_name": "Peer Name",
  "from_name": "My Name"
}
```

## 7. Group flow (to'liq)

1. Group identifikatsiyasi uchun `item_id` strategiyasini belgilang.
2. `create room` (`type: group`) yuboring.
3. Server roomni `project_id + type + item_id` bo'yicha tekshiradi.
4. Kerak bo'lsa room yaratadi.
5. Creator uchun `rooms list` yangilanadi.
6. Roomga kirish uchun `join room` yuboring.

Muhim:

- Group uchun ham real-time oqimda Socket `create room` afzal.
- REST `POST /v1/room` faqat controlled/backoffice ssenariyda ishlatiladi.

`create room` (group) payload:

```json
{
  "name": "Support Group",
  "type": "group",
  "row_id": "my-row-id",
  "project_id": "project-id",
  "item_id": "ticket-or-entity-id",
  "to_name": "Support Group"
}
```

## 8. Roomga kirish va history

1. User roomni bosadi.
2. `join room` emit qiling.
3. Inputlarni faollashtiring.
4. `room history` kelganda message listga yozing.
5. Shu roomdagi yangi `chat message` eventlarni append qiling.

`join room` payload:

```json
{
  "room_id": "room-uuid",
  "row_id": "my-row-id",
  "project_id": "project-id",
  "offset": 0,
  "limit": 100
}
```

## 9. Message yuborish

`chat message` payload:

```json
{
  "room_id": "room-uuid",
  "project_id": "project-id",
  "from": "My Name",
  "author_row_id": "my-row-id",
  "content": "Hello",
  "type": "text",
  "file": "",
  "parent_id": null
}
```

Qoidalar:

- `room_id`, `from`, `author_row_id` bo'sh bo'lmasin.
- Rasm yuborsangiz `type: image`, `file` ga URL bering.
- Optimistic UI ishlating, lekin serverdan kelgan message bilan reconcile qiling.

## 10. Read receipt flow

1. Chat ochilganda ko'rinib turgan message'lar uchun `message:read` yuboring.
2. `message.read` eventni tinglab UIda read holatni yangilang.

`message:read` payload:

```json
{
  "row_id": "my-row-id",
  "room_id": "room-uuid"
}
```

## 11. Typing flow

1. Inputga yozishni boshlaganda bir marta `typing:start` yuboring.
2. 1200-1500 ms idle bo'lsa `typing:stop` yuboring.
3. Send tugmasida ham `typing:stop` yuboring.
4. Boshqa userdan `typing:start/stop` kelganda indikator ko'rsating/yashiring.

Payload:

```json
{
  "room_id": "room-uuid",
  "user_name": "My Name",
  "row_id": "my-row-id",
  "project_id": "project-id"
}
```

## 12. Presence flow

1. `connected` yuborilganda online bo'ladi.
2. Har 30 sekundda `presence:ping` yuboring.
3. App yopilishida `disconnected` yuboring.
4. `presence.updated` event bilan user statusini yangilang.

`presence:ping` payload:

```json
{
  "row_id": "my-row-id",
  "project_id": "project-id"
}
```

## 13. Reconnect strategiya (juda muhim)

Reconnect bo'lganda quyidagini avtomat qiling:

1. `connected` qayta yuborish.
2. `rooms list` qayta so'rash.
3. Oxirgi ochiq room bo'lsa `join room` qayta yuborish.
4. Typing holatini reset qilish.

Tavsiya:

- `connected`ni idempotent deb qabul qiling.
- UIda connection banner ko'rsating (`Connected`/`Reconnecting`/`Offline`).
- Reconnectdan keyin `create room` qayta chaqirmang.
- Reconnectdan keyin faqat `rooms list` va kerak bo'lsa `join room` qiling.

## 14. Minimal Dart skeleton

```dart
class ChatSocketService {
  late Socket _socket;

  void connect({required String baseUrl}) {
    _socket = io(baseUrl, {
      'transports': ['websocket'],
      'reconnection': true,
      'reconnectionAttempts': double.infinity,
      'reconnectionDelay': 500,
    });

    _socket.onConnect((_) {
      // 1) connected emit
      // 2) heartbeat start
    });

    _socket.on('rooms list', (data) {
      // map -> Room list
    });
    _socket.on('room history', (data) {
      // map -> Message list
    });
    _socket.on('chat message', (data) {
      // append message
    });
    _socket.on('check room', (roomId) {
      // emit join room
    });
    _socket.on('error', (e) {
      // error handling
    });
  }

  void emitConnected(String rowId, String projectId) {
    _socket.emit('connected', {
      'row_id': rowId,
      'project_id': projectId,
      'offset': 0,
      'limit': 50,
    });
  }
}
```

## 15. UI sahifalar bo'yicha tavsiya

### Rooms Page

- Init: connect bo'lsa `rooms list`ni tinglang
- Pull-to-refresh: `rooms list` emit
- DM create tugma: `create room`
- Group create tugma: `create room (group)`

### Chat Page

- Open: `join room`
- Incoming: `chat message`
- Send: `chat message`
- Read: `message:read`
- Typing: `typing:start/stop`

## 16. Error handling qoidalari

- `error` event kelganda toast/snackbar chiqaring.
- Network xatoda retry tugma bering.
- Payload validation client tomonda ham qiling (`room_id`, `row_id`, `project_id`).

## 17. Testing checklist

1. DM create 2 marta bosilganda duplicate room bo'lmayaptimi.
2. Group create bir xil `item_id`da duplicate bo'lmayaptimi.
3. Reconnectdan keyin xabar kelishi tiklanadimi.
4. Read receipt ikkinchi userga boradimi.
5. Typing indicator stale qolmayaptimi.
6. Presence ping to'xtasa offlinega o'tadimi.

## 18. Production checklist

1. Auth tokenni socket handshakega qo'shing.
2. Message ACK/Retry qo'shing.
3. Offline queue qo'shing.
4. Lokal cache (room/message) qo'shing.
5. Crash-safe reconnect scenariylarini test qiling.

## 19. Tez-tez uchraydigan muammo va yechim

- Muammo: room yaratildi, lekin chat ochilmadi.
- Yechim: `check room` eventini tutib `join room` yuborayotganingizni tekshiring.

- Muammo: online status noto'g'ri.
- Yechim: `presence:ping` interval ishlayotganini va `project_id` yuborilayotganini tekshiring.

- Muammo: read ishlamayapti.
- Yechim: `message:read` payloadda `row_id` va `room_id` borligini tekshiring.


## 20. Request/Response Flow (nima qilsa nima bo'ladi)

Bu bo'lim Flutter implementatsiya paytida eng ko'p kerak bo'ladigan real ketma-ketlik.

### 20.1 Connect bo'lganda

Flutter nima qiladi:

```dart
socket.emit('connected', {
  'row_id': meRowId,
  'project_id': projectId,
  'offset': 0,
  'limit': 50,
});
```

Backend nima qiladi:

1. Socketni `row_id` roomiga join qiladi.
2. Presence'ni `online` qiladi.
3. Shu user uchun `rooms list`ni qaytaradi.

Flutter nimani kutadi:

```dart
socket.on('rooms list', (rooms) {
  // Rooms page state update
});

socket.on('presence.updated', (p) {
  // Online badge update
});
```

### 20.2 DM create qilganda

Flutter nima qiladi:

```dart
socket.emit('create room', {
  'row_id': meRowId,
  'project_id': projectId,
  'type': 'single',
  'to_row_id': peerRowId,
  'to_name': peerName,
  'from_name': meName,
});
```

Backend nima qiladi:

1. Oldin DM bor-yo'qligini tekshiradi.
2. Bor bo'lsa yangi room ochmaydi, `check room` qaytaradi.
3. Yo'q bo'lsa 1 ta room yaratadi va memberlar qo'shadi.
4. Creatorga `rooms list` yuboradi.

Flutter nimani kutadi:

```dart
socket.on('check room', (roomId) {
  socket.emit('join room', {
    'room_id': roomId,
    'row_id': meRowId,
    'project_id': projectId,
    'offset': 0,
    'limit': 100,
  });
});

socket.on('rooms list', (rooms) {
  // Rooms update
});
```

### 20.3 Join room qilganda

Flutter nima qiladi:

```dart
socket.emit('join room', {
  'room_id': roomId,
  'row_id': meRowId,
  'project_id': projectId,
  'offset': 0,
  'limit': 100,
});
```

Backend nima qiladi:

1. Socketni roomga join qiladi.
2. `room history` yuboradi.
3. `rooms list`ni ham refresh qilib yuboradi.

Flutter nimani kutadi:

```dart
socket.on('room history', (messages) {
  // Chat page initial list
});
```

### 20.4 Xabar yuborganda

Flutter nima qiladi:

```dart
socket.emit('chat message', {
  'room_id': roomId,
  'project_id': projectId,
  'from': meName,
  'author_row_id': meRowId,
  'content': text,
  'type': 'text',
  'file': '',
  'parent_id': null,
});
```

Backend nima qiladi:

1. Messageni DBga yozadi.
2. `chat message`ni roomga broadcast qiladi.
3. Room memberlariga `rooms list` push qiladi.

Flutter nimani kutadi:

```dart
socket.on('chat message', (msg) {
  // Append incoming message
});
```

### 20.5 Read yuborganda

Flutter nima qiladi:

```dart
socket.emit('message:read', {
  'row_id': meRowId,
  'room_id': roomId,
});
```

Backend nima qiladi:

1. `last_read_at` ni yangilaydi.
2. O'qilmagan xabarlar `read_at` ni belgilaydi.
3. `message.read`ni roomga yuboradi.

Flutter nimani kutadi:

```dart
socket.on('message.read', (event) {
  // mark seen in UI
});
```

## 21. REST curl + Flutter mapping

Socketdan tashqari, debugging uchun RESTni `curl` bilan tekshirishingiz mumkin.

Muhim cheklov:

- `GET /v1/room` so'roviga `to_row_id` qo'shish backendda DM filter bermaydi.
- DM ni aniq topish uchun `POST /v1/room/exist` yoki Socket `create room` + `check room` flow ishlating.

### 21.1 Room listni curl bilan tekshirish

```bash
BASE_URL="https://chat-service.u-code.io"
PROJECT_ID="aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
ME="11111111-1111-1111-1111-111111111111"

curl -sS "$BASE_URL/v1/room?row_id=$ME&project_id=$PROJECT_ID&offset=0&limit=20"
```

Flutter mapping:

```dart
final res = await dio.get(
  '$baseUrl/v1/room',
  queryParameters: {
    'row_id': meRowId,
    'project_id': projectId,
    'offset': 0,
    'limit': 20,
  },
);
final rooms = res.data['body']['rooms'];
```

### 21.2 Message historyni curl bilan tekshirish

```bash
ROOM_ID="room-uuid"
curl -sS "$BASE_URL/v1/message?room_id=$ROOM_ID&offset=0&limit=50"
```

Flutter mapping:

```dart
final res = await dio.get(
  '$baseUrl/v1/message',
  queryParameters: {
    'room_id': roomId,
    'offset': 0,
    'limit': 50,
  },
);
final messages = res.data['body']['messages'];
```

## 22. Copy-paste payloadlar

### 22.1 `create room` (DM)

```json
{
  "row_id": "11111111-1111-1111-1111-111111111111",
  "project_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "type": "single",
  "to_row_id": "22222222-2222-2222-2222-222222222222",
  "to_name": "Porthos",
  "from_name": "Athos"
}
```

### 22.2 `join room`

```json
{
  "room_id": "room-uuid",
  "row_id": "11111111-1111-1111-1111-111111111111",
  "project_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "offset": 0,
  "limit": 100
}
```

### 22.3 `chat message`

```json
{
  "room_id": "room-uuid",
  "project_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "from": "Athos",
  "author_row_id": "11111111-1111-1111-1111-111111111111",
  "content": "Salom",
  "type": "text",
  "file": "",
  "parent_id": null
}
```

### 22.4 `message:read`

```json
{
  "row_id": "11111111-1111-1111-1111-111111111111",
  "room_id": "room-uuid"
}
```

## 23. Logdan kelib chiqqan muhim xatolar va to'g'ri yechim

### 23.1 Har kirishda yangi room ochilib ketishi

Sabab:

- Client `create room`ni noto'g'ri joyda (masalan screen open payti) chaqiradi.
- Yoki REST `POST /v1/room`ni bevosita ishlatadi.

Yechim:

1. `create room`ni faqat user action (DM tugmasi)da chaqiring.
2. `create room`ni Socket orqali yuboring.
3. `check room` kelsa faqat `join room` qiling.

### 23.2 `row_id` almashib qolishi

Sabab:

- Bir user sessionida `row_id` bir necha qiymatga o'zgarib ketadi.

Yechim:

1. `ChatRepo.initialize`da `row_id`ni freeze qiling.
2. `configMatch` false bo'lsa socketni toza reconnect qiling.
3. Bitta login sessionida bitta `row_id` qat'iy saqlansin.

### 23.3 `join room` va `chat message room_id` mos emas

Sabab:

- UI state `activeRoomId` yangilanmagan yoki eski roomId qolib ketgan.

Yechim:

1. `join room` ACK/eventidan keyin `activeRoomId = joinedRoomId` set qiling.
2. `chat message` yuborishda faqat `activeRoomId`dan foydalaning.
3. Agar `activeRoomId == null` bo'lsa sendni bloklang.

### 23.4 `initialize complete, socketConnected=false`

Sabab:

- `initialize()` socket ulanishi tugashidan oldin yakunlangan (async race).

Yechim:

1. `connect()`dan keyin `onConnect` future/completer kuting.
2. `connected` event emitini faqat real socket connected bo'lganda qiling.

## 24. `attributes` bilan profil rasmi va metadata saqlash

Bu servisda `attributes` JSON ko'rinishida saqlanadi va profil rasmi, display name, qo'shimcha chat metadata uchun ishlatish mumkin.

### 24.1 Qayerda saqlanadi

- Room level: `rooms.attributes`
- Member level: `room_members.attributes`

Socket `create room`da quyidagi keylar ishlaydi:

- `attributes` -> room attributes
- `member_attributes` -> creator member attributes
- `to_member_attributes` -> ikkinchi user member attributes (`single` holatda)

### 24.2 Tavsiya etilgan room-level schema (profil uchun)

```json
{
  "profiles": {
    "my-row-id": {
      "name": "Ali",
      "pic": "https://cdn.example.com/u/ali.jpg"
    },
    "peer-row-id": {
      "name": "Doktor D",
      "pic": "https://cdn.example.com/u/doktor-d.jpg"
    }
  },
  "meta": {
    "chat_title": "Doktor bilan chat",
    "source": "appointment"
  }
}
```

Bu schema bilan room list yoki chat headerda peer rasmi va ismini `profiles[peerRowId]` orqali olasiz.

### 24.3 DM yaratishda attributes yuborish (Socket)

```dart
socket.emit('create room', {
  'row_id': meRowId,
  'project_id': projectId,
  'type': 'single',
  'to_row_id': peerRowId,
  'to_name': peerName,
  'from_name': meName,
  'attributes': {
    'profiles': {
      meRowId: {'name': meName, 'pic': mePicUrl},
      peerRowId: {'name': peerName, 'pic': peerPicUrl},
    },
    'meta': {'source': 'appointment'}
  },
  'member_attributes': {
    'role': 'owner',
    'muted': false
  },
  'to_member_attributes': {
    'role': 'participant',
    'muted': false
  }
});
```

### 24.4 Group yaratishda attributes yuborish

```dart
socket.emit('create room', {
  'name': 'Support Group',
  'type': 'group',
  'row_id': meRowId,
  'project_id': projectId,
  'item_id': itemId,
  'attributes': {
    'avatar': 'https://cdn.example.com/group/support.png',
    'topic': 'Support',
    'profiles': {
      meRowId: {'name': meName, 'pic': mePicUrl}
    }
  }
});
```

### 24.5 Flutterda o'qish (safe parse)

```dart
Map<String, dynamic> attrs = {};
final raw = room['attributes'];
if (raw is Map<String, dynamic>) attrs = raw;

final profiles = (attrs['profiles'] as Map?) ?? {};
final peer = profiles[peerRowId] as Map?;
final peerPic = (peer?['pic'] as String?) ?? '';
final peerName = (peer?['name'] as String?) ?? (room['to_name'] ?? '');
```

### 24.6 Amaliy tavsiyalar

1. Rasmni `base64` emas, URL saqlang.
2. `attributes` ichiga katta payload (masalan full profile object) solmang.
3. `profiles` mapda key sifatida har doim `row_id` ishlating.
4. `from_name` ni bo'sh yubormang, aks holda ikkinchi member metadata to'liq tushmasligi mumkin.
5. Schema version qo'shish foydali: `\"schema_version\": 1`.


## 25. Flutter Full Feature Guide (DM + Group + Member Management + History)

Bu bo'lim Flutter jamoasi uchun yakuniy integratsiya qo'llanmasi.

### 25.1 Feature matrix (hozirgi backend holati)

- DM create/open: bor
- Group create/open: bor
- Group member add: bor (`POST /v1/room-member`)
- Group member remove: public API yo'q (backendda remove endpoint yo'q)
- History list: bor (`join room` yoki `GET /v1/message`)
- History pagination: bor (`offset`, `limit`)
- Send text/image: bor (`chat message`)
- Read receipt: bor (`message:read`)
- Typing indicator: bor (`typing:start/typing:stop`)
- Presence online/offline: bor (`connected`, `presence:ping`, sweeper)

### 25.2 DM full flow (production)

1. Screen open -> socket ready bo'lsa `connected` yuboriladi.
2. DM open action -> `socket create room` yuboriladi.
3. `check room` bo'lsa `join room` yuboriladi.
4. `room history` keladi.
5. Send -> `chat message`.
6. Chat visible bo'lsa -> `message:read`.

Qoidalar:

- DM uchun `from_name` bo'sh bo'lmasin.
- Har open payti `POST /v1/room` qilmang.

### 25.3 Group full flow

1. Group create action -> `socket create room` (`type=group`, `item_id`).
2. Group room card open -> `join room`.
3. History yuklanadi (`room history`).
4. Send/read/typing DM bilan bir xil.

### 25.4 Group member add flow

Public API:

- `POST /v1/room-member`

Flutter misol:

```dart
await dio.post(
  '$baseUrl/v1/room-member',
  data: {
    'room_id': roomId,
    'row_id': newMemberRowId,
    'to_name': newMemberName,
    'to_row_id': newMemberRowId,
    'attributes': {
      'role': 'member',
      'added_by': myRowId,
    }
  },
);
```

Keyin nima qilish kerak:

1. Member qo'shilgach clientda `rooms list` so'rang yoki socketdan kuting.
2. Yangi member tomonda socket ulanishidan keyin room ro'yxatida ko'rinadi.

### 25.5 Group member remove flow (muhim cheklov)

Hozirgi backendda:

- `DELETE /v1/room-member` yoki `remove member` socket event yo'q.

Demak to'g'ridan-to'g'ri remove qilish public contractda yo'q.

Vaqtinchalik workaround variantlar:

1. UI-level hide: memberni lokal ro'yxatdan yashirish (faqat presentation).
2. Role-based block: `attributes` orqali `is_removed=true` flag saqlab, clientda yuborishni bloklash.
3. Backend extension: alohida remove endpoint qo'shish (eng to'g'ri yechim).

Tavsiya:

- Productionda haqiqiy remove kerak bo'lsa backendga `DELETE /v1/room-member` qo'shilsin.

### 25.6 History strategy (initial + pagination)

Initial load:

- `join room` -> server `room history` qaytaradi (`offset=0`, `limit=50/100`).

Older messages pagination:

- REST: `GET /v1/message?room_id=...&offset=50&limit=50`
- Yoki socket `room history`ni boshqa offset bilan yuborish.

Flutter merge qoidasi:

1. Yangi page kelganda duplicate `id`larni olib tashlang.
2. Messagesni `created_at` bo'yicha ascending tuting.
3. `isLoadingMore` flag bilan parallel paginationni bloklang.

### 25.7 Unread + read behavior

- Room listda `unread_message_count` keladi.
- Chat ochilganda visible xabarlar uchun `message:read` yuboring.
- `message.read` event kelganda message statusni seen qilib yangilang.

### 25.8 Message send safety

Senddan oldin tekshiruv:

1. `activeRoomId != null`
2. `socket.connected == true`
3. `author_row_id` to'g'ri (`currentUser.rowId`)

Aks holda sendni bloklang.

### 25.9 Multi-room bugni oldini olish

1. `join room`dan keyin faqat o'sha `roomId` ni `activeRoomId`ga set qiling.
2. `chat message` emitda doim `activeRoomId` ishlating.
3. Screen dispose bo'lsa typing timerlarni tozalang.

### 25.10 Minimal repository flow pseudocode

```dart
Future<String> openDm({required String peerRowId, required String peerName}) async {
  await ensureSocketReady();

  // Preferred: socket duplicate-safe flow
  socket.emit('create room', {
    'row_id': meRowId,
    'project_id': projectId,
    'type': 'single',
    'to_row_id': peerRowId,
    'to_name': peerName,
    'from_name': meName,
  });

  final roomId = await waitCheckRoomOrResolveFromRoomsList();
  await joinRoom(roomId);
  return roomId;
}
```

### 25.11 Qaysi API qachon

- Real-time chat: Socket
- Debug/manual sync: REST
- Member add: REST (`POST /v1/room-member`)
- Member remove: hozircha yo'q (backend extension kerak)

### 25.12 Final integration checklist

1. `row_id` session davomida o'zgarmaydi.
2. `from_name` doim yuboriladi.
3. `create room` only-on-action (auto emas).
4. `check room` handler yozilgan.
5. `activeRoomId` strict boshqariladi.
6. Pagination duplicate-safe merge bilan ishlaydi.
7. Reconnectdan keyin create emas, join/list ishlatiladi.
8. `attributes.profiles[row_id]` bilan avatar/name chiqariladi.

## 26. Last Online (`last_seen_at`) ni Flutterda ko'rsatish

Bu bo'lim user qachon online bo'lganini (yoki oxirgi marta qachon ko'rilganini) ko'rsatish uchun.

### 26.1 Qaysi eventdan olinadi

- `presence.updated` eventida `row_id`, `status`, `last_seen_at` keladi.
- `presence:get` yuborsangiz, javob ham `presence.updated` eventida keladi.

`presence:get` payload:

```json
{
  "row_id": "peer-row-id"
}
```

### 26.2 Socket listener (Flutter)

```dart
socket.on('presence.updated', (p) {
  final rowId = p['row_id'] as String?;
  final status = p['status'] as String?; // online | offline
  final lastSeenRaw = p['last_seen_at'] as String?;
  if (rowId == null) return;

  final lastSeen = _parseServerTime(lastSeenRaw);
  presenceStore[rowId] = PresenceUi(
    status: status ?? 'offline',
    lastSeenAt: lastSeen,
  );
});
```

### 26.3 Server vaqt formatini parse qilish

Backend ko'pincha RFC1123 format qaytaradi (`Thu, 26 Mar 2026 10:50:58 UTC`).

```dart
DateTime? _parseServerTime(String? raw) {
  if (raw == null || raw.isEmpty) return null;
  try {
    // intl package bilan parse qilish tavsiya etiladi.
    final dt = DateFormat("EEE, dd MMM yyyy HH:mm:ss 'UTC'", 'en_US').parseUtc(raw);
    return dt.toLocal();
  } catch (_) {
    return null;
  }
}
```

### 26.4 UI text (`online` / `last seen ...`)

```dart
String buildPresenceLabel(PresenceUi? p) {
  if (p == null) return 'Unknown';
  if (p.status == 'online') return 'Online';
  if (p.lastSeenAt == null) return 'Offline';
  return 'Last seen ${timeAgo(p.lastSeenAt!)}';
}
```

`timeAgo` uchun oddiy misol:

```dart
String timeAgo(DateTime t) {
  final d = DateTime.now().difference(t);
  if (d.inSeconds < 60) return '${d.inSeconds}s ago';
  if (d.inMinutes < 60) return '${d.inMinutes}m ago';
  if (d.inHours < 24) return '${d.inHours}h ago';
  return '${d.inDays}d ago';
}
```

### 26.5 Qachon `presence:get` yuborish kerak

1. Chat header ochilganda peer statusni tez olish uchun.
2. App resume bo'lganda statusni refresh qilish uchun.
3. Reconnectdan keyin roomdagi userlar uchun batch tarzda (navbat bilan).

### 26.6 Muhim cheklov

- Hozirgi backendda presence `row_id` bo'yicha global.
- `project_id` kesimida alohida last online saqlanmaydi.

