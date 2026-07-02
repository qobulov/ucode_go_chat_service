package models

import (
	"encoding/json"
	"time"
)

type Message struct {
	Id             string          `json:"id"`
	RoomId         string          `json:"room_id"`
	Message        string          `json:"message"`
	Type           string          `json:"type"`
	File           string          `json:"file"`
	AuthorRowId    string          `json:"author_row_id"`
	From           string          `json:"from"`
	ParentId       any             `json:"parent_id"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
	ReadAt         string          `json:"read_at"`
	UserAttributes json.RawMessage `json:"attributes"`
}

type CreateMessage struct {
	RoomId      string `json:"room_id"`
	Message     string `json:"message"`
	AuthorRowId string `json:"author_row_id"`
	From        string `json:"from"`
	Type        string `json:"type"`
	File        string `json:"file"`
	ParentId    any    `json:"parent_id"`
}

type MarkReadMessage struct {
	RoomId      string    `json:"room_id"`
	ReaderRowId string    `json:"row_id"`
	ReadAt      time.Time `json:"read_at"`
}

type MarkReadMessageResp struct {
	RoomId  string `json:"room_id"`
	ReadAt  string `json:"read_at"`
	Updated bool   `json:"updated"`
}

type GetListMessageReq struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	RoomId string `json:"room_id"`
}

type GetListMessageResp struct {
	Count    uint64     `json:"count"`
	Messages []*Message `json:"messages"`
}
