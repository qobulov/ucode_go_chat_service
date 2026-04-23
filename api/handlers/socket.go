package handlers

import (
	"encoding/json"
	"sync"
	"time"
	"ucode/ucode_go_chat_service/config"
	"ucode/ucode_go_chat_service/internal/socketio"
	"ucode/ucode_go_chat_service/models"
	"ucode/ucode_go_chat_service/pkg/logger"
	"ucode/ucode_go_chat_service/pkg/utils"
	"ucode/ucode_go_chat_service/storage"

	"github.com/spf13/cast"
)

type socket struct {
	mu        *sync.RWMutex
	io        *socketio.Io
	storage   storage.StorageI
	log       *logger.Logger
	socketRow map[string]string
}

func NewSocketHandler(io *socketio.Io, strg storage.StorageI, log *logger.Logger) {
	s := &socket{
		mu:      &sync.RWMutex{},
		io:      io,
		storage: strg,
		log:     log,
	}

	io.OnAuthentication(func(params map[string]string) bool {
		return true
	})

	io.OnConnection(func(sk *socketio.Socket) {
		sk.On("connected", s.onConnection)

		sk.On("create room", s.onCreateRoom)
		sk.On("join room", s.onJoinRoom)
		sk.On("rooms list", s.onRoomsList)
		sk.On("chat message", s.onChatMessage)
		sk.On("room history", s.onRoomHistory)

		sk.On("presence:connected", s.onPresenceConnected)
		sk.On("presence:ping", s.onPresencePing)
		sk.On("presence:get", s.onPresenceGet)

		sk.On("message:read", s.onMessageRead)
		sk.On("message:update", s.onMessageUpdate)

		sk.On("typing:start", s.onTypingStart)
		sk.On("typing:stop", s.onTypingStop)

		sk.On("room:delete", s.onRoomDelete)

		sk.On("disconnected", s.onDisconnection)
	})
}

func (s *socket) onConnection(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		err := "payload is not provided"
		s.emitErr(event.Socket, sockErr{Function: "onConnection", Message: err, Error: err})
		return
	}
	params := utils.ConvertMaptoStruct[models.Connection](reqMap)
	if params.RowId == "" {
		err := "rowId is required"
		s.emitErr(event.Socket, sockErr{Function: "onConnection", Message: err, Error: err, Request: reqMap})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	event.Socket.Join(params.RowId)

	ctx, cancel := s.ctx()
	defer cancel()

	now := time.Now().UTC()
	if err := s.storage.Postgres().PresenceUpsert(ctx, &models.UpsertPresence{
		RowId:     params.RowId,
		ProjectId: params.ProjectId,
		Status:    "online",
		Now:       now,
	}); err != nil {
		errMsg := "failed to update presence"
		s.emitErr(event.Socket, sockErr{Function: "onConnection", Message: errMsg, Error: err.Error(), Request: reqMap})
		return
	}

	s.io.Emit("presence.updated", map[string]any{
		"row_id":       params.RowId,
		"status":       "online",
		"last_seen_at": now,
	})

	reqType := ""
	if typeVal, ok := reqMap["type"].(string); ok {
		reqType = typeVal
	}

	items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
		RowId:     params.RowId,
		Type:      reqType,
		ProjectId: params.ProjectId,
		Offset:    params.Offset,
		Limit:     params.Limit,
		Search:    params.Search,
	})
	if err != nil {
		errMsg := "failed to load rooms"
		s.emitErr(event.Socket, sockErr{Function: "onConnection", Message: errMsg, Error: err.Error(), Request: reqMap})
		return
	}

	event.Socket.Emit("rooms list", items.Rooms)
}

func (s *socket) onCreateRoom(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		err := "invalid payload"
		s.emitErr(event.Socket, sockErr{Function: "onCreateRoom", Message: err, Error: err})
		return
	}
	params := utils.ConvertMaptoStruct[models.CreateRoom](reqMap)

	if params.RowId == "" || params.ProjectId == "" || params.Type == "" {
		s.emitErr(event.Socket, sockErr{Function: "onCreateRoom", Message: "rowId, projectId and type are required"})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	ctx, cancel := s.ctx()
	defer cancel()

	id, err := s.storage.Postgres().RoomExists(ctx, &models.ExistsRoom{
		Type:      params.Type,
		ProjectId: params.ProjectId,
		RowId:     params.RowId,
		ToRowId:   params.ToRowId,
		ItemId:    params.ItemId,
	})
	if err == nil && id != "" {
		memberAttributes := extractAttributes(reqMap, "member_attributes")
		if len(memberAttributes) == 0 {
			memberAttributes = extractAttributes(reqMap, "attributes")
		}
		_, _ = s.storage.Postgres().RoomMemberCreate(ctx, &models.CreateRoomMember{
			RoomId:     id,
			RowId:      params.RowId,
			ToName:     params.ToName,
			ToRowId:    params.ToRowId,
			Attributes: memberAttributes,
		})
		event.Socket.Emit("check room", id)

		reqType := ""
		if typeVal, ok := reqMap["type"].(string); ok {
			reqType = typeVal
		}

		items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
			RowId:     params.RowId,
			Type:      reqType,
			ProjectId: params.ProjectId,
			Offset:    params.Offset,
			Limit:     params.Limit,
		})
		if err == nil {
			event.Socket.Emit("rooms list", items.Rooms)
		}
		return
	}

	room, err := s.storage.Postgres().RoomCreate(ctx, &params)
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onCreateRoom", Message: "failed to create room", Error: err.Error(), Request: reqMap})
		return
	}

	memberAttributes := extractAttributes(reqMap, "member_attributes")
	if len(memberAttributes) == 0 {
		memberAttributes = extractAttributes(reqMap, "attributes")
	}
	_, err = s.storage.Postgres().RoomMemberCreate(ctx, &models.CreateRoomMember{
		RoomId:     room.Id,
		RowId:      params.RowId,
		ToName:     params.ToName,
		ToRowId:    params.ToRowId,
		Attributes: memberAttributes,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onCreateRoom", Message: "failed to add member", Error: err.Error(), Request: reqMap})
		return
	}

	toRowId := cast.ToString(params.ToRowId)

	if params.ToRowId != "" && params.FromName != "" && params.Type == "single" {
		toMemberAttributes := extractAttributes(reqMap, "to_member_attributes")
		_, err = s.storage.Postgres().RoomMemberCreate(ctx, &models.CreateRoomMember{
			RoomId:     room.Id,
			RowId:      toRowId,
			ToName:     params.FromName,
			ToRowId:    params.RowId,
			Attributes: toMemberAttributes,
		})
		if err != nil {
			s.emitErr(event.Socket, sockErr{Function: "onCreateRoom", Message: "failed to add member 2", Error: err.Error(), Request: reqMap})
			return
		}
	}

	reqType := ""
	if typeVal, ok := reqMap["type"].(string); ok {
		reqType = typeVal
	}

	items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
		RowId:     params.RowId,
		Type:      reqType,
		ProjectId: params.ProjectId,
		Offset:    params.Offset,
		Limit:     params.Limit,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onCreateRoom", Message: "failed to fetch rooms list", Error: err.Error(), Request: reqMap})
		return
	}
	event.Socket.Emit("rooms list", items.Rooms)
}

func (s *socket) onJoinRoom(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onJoinRoom", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.JoinRoom](reqMap)

	if params.RoomId == "" || params.RowId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onJoinRoom", Message: "roomId and rowId are required"})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	ctx, cancel := s.ctx()
	defer cancel()

	_ = s.storage.Postgres().UpdateLastReadAt(ctx, &models.UpdateLastReadAtReq{
		RoomId: params.RoomId,
		RowId:  params.RowId,
	})

	item, err := s.storage.Postgres().RoomGetSingle(ctx, &models.GetSingleRoom{
		Id: params.RoomId,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onJoinRoom", Message: "room not found", Error: err.Error(), Request: reqMap})
		return
	}

	memberAttributes := extractAttributes(reqMap, "attributes")
	_, _ = s.storage.Postgres().RoomMemberCreate(ctx, &models.CreateRoomMember{
		RoomId:     item.Id,
		RowId:      params.RowId,
		ToName:     params.ToName,
		ToRowId:    params.ToRowId,
		Attributes: memberAttributes,
	})

	event.Socket.Join(item.Id)

	messageHistory, err := s.storage.Postgres().MessageGetList(ctx, &models.GetListMessageReq{
		RoomId: item.Id,
		Offset: params.Offset,
		Limit:  params.Limit,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onJoinRoom", Message: "failed to load history", Error: err.Error(), Request: reqMap})
		return
	}
	if len(messageHistory.Messages) > 0 {
		event.Socket.Emit("room history", messageHistory.Messages)
	}

	reqType := ""
	if typeVal, ok := reqMap["type"].(string); ok {
		reqType = typeVal
	}

	items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
		RowId:     params.RowId,
		Type:      reqType,
		ProjectId: params.ProjectId,
		Offset:    params.Offset,
		Limit:     params.Limit,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onJoinRoom", Message: "failed to load rooms list", Error: err.Error(), Request: reqMap})
		return
	}
	event.Socket.Emit("rooms list", items.Rooms)
}

func (s *socket) onRoomsList(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onRoomsList", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.RoomsList](reqMap)

	if params.RowId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onRoomsList", Message: "rowId is required"})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	ctx, cancel := s.ctx()
	defer cancel()

	items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
		RowId:     params.RowId,
		Type:      params.Type,
		ProjectId: params.ProjectId,
		Offset:    params.Offset,
		Limit:     params.Limit,
		Search:    params.Search,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onRoomsList", Message: "failed to load rooms", Error: err.Error(), Request: reqMap})
		return
	}

	event.Socket.Emit("rooms list", items.Rooms)
}

func (s *socket) onRoomHistory(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onRoomHistory", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.RoomHistoryReq](reqMap)
	if params.RoomId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onRoomHistory", Message: "room id is required"})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	ctx, cancel := s.ctx()
	defer cancel()

	messageHistory, err := s.storage.Postgres().MessageGetList(ctx, &models.GetListMessageReq{
		RoomId: params.RoomId,
		Offset: params.Offset,
		Limit:  params.Limit,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onRoomHistory", Message: "failed to load history", Error: err.Error(), Request: reqMap})
		return
	}

	event.Socket.Emit("room history", messageHistory.Messages)
}

func (s *socket) onChatMessage(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onChatMessage", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.ChatMessage](reqMap)

	if params.RoomId == "" || params.From == "" || params.AuthorRowId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onChatMessage", Message: "missing required fields"})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	ctx, cancel := s.ctx()
	defer cancel()

	_ = s.storage.Postgres().UpdateLastReadAt(ctx, &models.UpdateLastReadAtReq{
		RoomId: params.RoomId,
		RowId:  params.AuthorRowId,
	})

	message, err := s.storage.Postgres().MessageCreate(ctx, &models.CreateMessage{
		RoomId:      params.RoomId,
		Message:     params.Content,
		From:        params.From,
		Type:        params.Type,
		File:        params.File,
		AuthorRowId: params.AuthorRowId,
		ParentId:    params.ParentId,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onChatMessage", Message: "failed to send", Error: err.Error(), Request: reqMap})
		return
	}

	s.io.To(params.RoomId).Emit("chat message", message)

	members, err := s.storage.Postgres().RoomMembersByRoomId(ctx, params.RoomId)
	if err == nil {
		for _, m := range members {
			items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
				RowId:     m.RowId,
				ProjectId: params.ProjectId,
				Offset:    params.Offset,
				Limit:     params.Limit,
			})
			if err == nil {
				s.io.To(m.RowId).Emit("rooms list", items.Rooms)
			}
		}
	}
}

func (s *socket) onPresenceConnected(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onPresenceConnected", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.PresenceConnected](reqMap)

	if params.RowId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onPresenceConnected", Message: "row_id is required"})
		return
	}

	event.Socket.Join(params.RowId)

	ctx, cancel := s.ctx()
	defer cancel()

	now := time.Now().UTC()

	err := s.storage.Postgres().PresenceUpsert(ctx, &models.UpsertPresence{
		RowId:     params.RowId,
		ProjectId: params.ProjectId,
		Status:    "online",
		Now:       now,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onPresenceConnected", Message: "failed to update presence", Error: err.Error(), Request: reqMap})
		return
	}

	s.io.Emit("presence.updated", map[string]any{
		"row_id":       params.RowId,
		"status":       "online",
		"last_seen_at": now,
	})
}

func (s *socket) onPresencePing(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onPresencePing", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.PresencePing](reqMap)
	if params.RowId == "" || params.ProjectId == "" {
		return
	}

	ctx, cancel := s.ctx()
	defer cancel()

	now := time.Now().UTC()

	_ = s.storage.Postgres().PresenceHeartbeat(ctx, &models.HeartbeatPresence{
		ProjectId: params.ProjectId,
		RowId:     params.RowId,
		Now:       now,
	})

	s.io.Emit("presence.updated", map[string]any{
		"row_id":       params.RowId,
		"status":       "online",
		"last_seen_at": now,
		"project_id":   params.ProjectId,
	})
}

func (s *socket) onPresenceGet(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onPresenceGet", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.PresenceGet](reqMap)
	if params.RowId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onPresenceGet", Message: "row_id is required"})
		return
	}

	ctx, cancel := s.ctx()
	defer cancel()

	pr, err := s.storage.Postgres().PresenceGet(ctx, &models.GetPresence{
		RowId: params.RowId,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onPresenceGet", Message: "failed to get presence", Error: err.Error(), Request: reqMap})
		return
	}

	event.Socket.Emit("presence.updated", pr)
}

func (s *socket) onMessageRead(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onMessageRead", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.ReadMessage](reqMap)
	if params.RowId == "" || params.RoomId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onMessageRead", Message: "row_id and room_id are required"})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	ctx, cancel := s.ctx()
	defer cancel()

	_ = s.storage.Postgres().UpdateLastReadAt(ctx, &models.UpdateLastReadAtReq{
		RoomId: params.RoomId,
		RowId:  params.RowId,
	})

	resp, err := s.storage.Postgres().MessageMarkRead(ctx, &models.MarkReadMessage{
		RoomId:      params.RoomId,
		ReaderRowId: params.RowId,
		ReadAt:      time.Now().UTC(),
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onMessageRead", Message: "failed to mark read", Error: err.Error(), Request: reqMap})
		return
	}
	if !resp.Updated {
		return
	}

	s.io.To(resp.RoomId).Emit("message.read", map[string]any{
		"room_id": resp.RoomId,
		"by":      params.RowId,
		"read_at": resp.ReadAt,
	})

	room, err := s.storage.Postgres().RoomGetSingle(ctx, &models.GetSingleRoom{Id: resp.RoomId})
	if err != nil {
		return
	}

	items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
		RowId:     params.RowId,
		ProjectId: room.ProjectId,
		Limit:     params.Limit,
	})
	if err == nil {
		event.Socket.Emit("rooms list", items.Rooms)
	}
}

func (s *socket) onMessageUpdate(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onMessageUpdate", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.UpdateMessage](reqMap)
	if params.Id == "" {
		s.emitErr(event.Socket, sockErr{Function: "onMessageUpdate", Message: "id is required"})
		return
	}

	ctx, cancel := s.ctx()
	defer cancel()

	resp, err := s.storage.Postgres().MessageUpdate(ctx, &params)
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onMessageUpdate", Message: "failed to update message", Error: err.Error(), Request: reqMap})
		return
	}

	event.Socket.Emit("message.update", resp)
}

func (s *socket) onDisconnection(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onDisconnection", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.Disconnection](reqMap)
	if params.RowId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onDisconnection", Message: "rowId is required"})
		return
	}

	ctx, cancel := s.ctx()
	defer cancel()

	now := time.Now().UTC()

	err := s.storage.Postgres().PresenceUpsert(ctx, &models.UpsertPresence{
		RowId:     params.RowId,
		ProjectId: params.ProjectId,
		Status:    "offline",
		Now:       now,
	})
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onDisconnection", Message: "failed to update presence", Error: err.Error(), Request: reqMap})
		return
	}

	s.io.Emit("presence.updated", map[string]any{
		"row_id":       params.RowId,
		"status":       "offline",
		"last_seen_at": now,
	})
}

func extractAttributes(reqMap map[string]any, key string) json.RawMessage {
	if attrs, ok := reqMap[key]; ok {
		if attrsMap, ok := attrs.(map[string]any); ok {
			data, err := json.Marshal(attrsMap)
			if err == nil {
				return data
			}
		} else if attrsStr, ok := attrs.(string); ok {
			if json.Valid([]byte(attrsStr)) {
				return json.RawMessage(attrsStr)
			}
		} else if attrsBytes, ok := attrs.([]byte); ok {
			if json.Valid(attrsBytes) {
				return json.RawMessage(attrsBytes)
			}
		}
	}
	return json.RawMessage("{}")
}

func (s *socket) onTypingStart(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		return
	}
	roomId, _ := reqMap["room_id"].(string)
	if roomId == "" {
		return
	}

	rowId, _ := reqMap["row_id"].(string)
	projectId, _ := reqMap["project_id"].(string)
	if rowId != "" && projectId != "" {
		ctx, cancel := s.ctx()
		defer cancel()
		now := time.Now().UTC()
		_ = s.storage.Postgres().PresenceHeartbeat(ctx, &models.HeartbeatPresence{
			RowId:     rowId,
			ProjectId: projectId,
			Now:       now,
		})
		s.io.Emit("presence.updated", map[string]any{
			"row_id":       rowId,
			"status":       "online",
			"last_seen_at": now,
			"project_id":   projectId,
		})
	}

	s.io.To(roomId).Emit("typing:start", reqMap)
}

func (s *socket) onRoomDelete(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		s.emitErr(event.Socket, sockErr{Function: "onRoomDelete", Message: "invalid payload"})
		return
	}
	params := utils.ConvertMaptoStruct[models.DeleteRoom](reqMap)

	if params.RoomId == "" || params.RowId == "" {
		s.emitErr(event.Socket, sockErr{Function: "onRoomDelete", Message: "room_id and row_id are required"})
		return
	}

	if params.Limit == 0 || params.Limit > config.DefaultRoomsLimit {
		params.Limit = config.DefaultRoomsLimit
	}

	ctx, cancel := s.ctx()
	defer cancel()

	members, err := s.storage.Postgres().RoomMembersByRoomId(ctx, params.RoomId)
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onRoomDelete", Message: "failed to get members", Error: err.Error(), Request: reqMap})
		return
	}

	err = s.storage.Postgres().RoomDelete(ctx, params.RoomId)
	if err != nil {
		s.emitErr(event.Socket, sockErr{Function: "onRoomDelete", Message: "failed to delete room", Error: err.Error(), Request: reqMap})
		return
	}

	s.io.To(params.RoomId).Emit("room.deleted", map[string]any{
		"room_id": params.RoomId,
		"by":      params.RowId,
	})

	for _, m := range members {
		items, err := s.storage.Postgres().RoomGetList(ctx, &models.GetListRoomReq{
			RowId:     m.RowId,
			ProjectId: params.ProjectId,
			Limit:     params.Limit,
		})
		if err == nil {
			s.io.To(m.RowId).Emit("rooms list", items.Rooms)
		}
	}
}

func (s *socket) onTypingStop(event *socketio.EventPayload) {
	reqMap, ok := event.Data[0].(map[string]any)
	if !ok {
		return
	}
	roomId, _ := reqMap["room_id"].(string)
	if roomId == "" {
		return
	}
	// Broadcast typing stop to everyone in the room
	s.io.To(roomId).Emit("typing:stop", reqMap)
}
