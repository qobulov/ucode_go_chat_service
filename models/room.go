package models

import "encoding/json"

type Room struct {
	Id                     string          `json:"id"`
	Name                   string          `json:"name"`
	Type                   string          `json:"type"`
	ProjectId              string          `json:"project_id"`
	ToName                 string          `json:"to_name"`
	ToRowId                any             `json:"to_row_id"`
	ItemId                 any             `json:"item_id"`
	Attributes             json.RawMessage `json:"attributes"`
	CreatedAt              string          `json:"created_at"`
	UpdatedAt              string          `json:"updated_at"`
	LastMessage            string          `json:"last_message,omitempty"`
	LastMessageType        string          `json:"last_message_type,omitempty"`
	LastMessageFile        string          `json:"last_message_file,omitempty"`
	LastMessageFrom        string          `json:"last_message_from,omitempty"`
	LastMessageCreatedAt   string          `json:"last_message_created_at"`
	UnreadMessageCount     int64           `json:"unread_message_count"`
	UserPresenceStatus     string          `json:"user_presence_status"`
	UserPresenceLastSeenAt string          `json:"user_presence_last_seen"`
}

type GetSingleRoom struct {
	Id string `json:"id"`
}

type CreateRoom struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	ProjectId  string          `json:"project_id"`
	RowId      string          `json:"row_id"`
	ToName     string          `json:"to_name"`
	ToRowId    any             `json:"to_row_id"`
	FromName   string          `json:"from_name"`
	ItemId     any             `json:"item_id"`
	Attributes json.RawMessage `json:"attributes"`
	Offset     uint64          `json:"offset"`
	Limit      uint64          `json:"limit"`
}

type UpdateRoom struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	ProjectId string `json:"project_id"`
}

type GetListRoomReq struct {
	Offset    uint64 `json:"offset"`
	Limit     uint64 `json:"limit"`
	RowId     string `json:"row_id"`
	Type      string `json:"type"`
	ProjectId string `json:"project_id"`
}

type GetListRoomResp struct {
	Count uint64  `json:"count"`
	Rooms []*Room `json:"rooms"`
}

type ExistsRoom struct {
	Type      string `json:"type"`
	ProjectId string `json:"project_id"`
	RowId     string `json:"row_id"`
	ToRowId   any    `json:"to_row_id"`
	ItemId    any    `json:"item_id"`
}

type UnreadCountReq struct {
	RoomId string `json:"room_id"`
	RowId  string `json:"row_id"`
}

type UnreadCountResp struct {
	RoomId string `json:"room_id"`
	RowId  string `json:"row_id"`
	Count  int64  `json:"count"`
}

type RoomHistoryReq struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	RoomId string `json:"room_id"`
}

type GetRoomIdByItemIdReq struct {
	ItemId    string `json:"item_id"`
	ProjectId string `json:"project_id"`
}

type GetRoomIdByItemIdResp struct {
	RoomId string `json:"room_id"`
}
