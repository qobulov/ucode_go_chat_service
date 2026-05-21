package models

import "encoding/json"

type RoomMember struct {
	Id         string          `json:"id"`
	RoomId     string          `json:"room_id"`
	RowId      string          `json:"row_id"`
	ToName     string          `json:"to_name"`
	ToRowId    any             `json:"to_row_id"`
	Attributes json.RawMessage `json:"attributes"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
}

type CreateRoomMember struct {
	RoomId     string          `json:"room_id"`
	RowId      string          `json:"row_id"`
	ToName     string          `json:"to_name"`
	ToRowId    any             `json:"to_row_id"`
	Attributes json.RawMessage `json:"attributes"`
}

type UpdateLastReadAtReq struct {
	RoomId string `json:"room_id"`
	RowId  string `json:"row_id"`
}

type UpdateRoomMember struct {
	RoomId     string          `json:"room_id"`
	RowId      string          `json:"row_id"`
	ToName     string          `json:"to_name"`
	ToRowId    any             `json:"to_row_id"`
	Attributes json.RawMessage `json:"attributes"`
}
