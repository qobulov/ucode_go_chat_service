package models

type Connection struct {
	RowId     string `json:"row_id"`
	Type      string `json:"type"`
    ProjectId string `json:"project_id"`
	Offset    uint64 `json:"offset"`
	Limit     uint64 `json:"limit"`
}

type Disconnection struct {
	RowId     string `json:"row_id"`
	ProjectId string `json:"project_id"`
}

type JoinRoom struct {
	RoomId  string `json:"room_id"`
	RowId   string `json:"row_id"`
	Type    string `json:"type"`
	ToName  string `json:"to_name"`
	ToRowId any    `json:"to_row_id"`
	Offset  uint64 `json:"offset"`
	Limit   uint64 `json:"limit"`
}

type ChatMessage struct {
	Content     string `json:"content"`
	RoomId      string `json:"room_id"`
	From        string `json:"from"`
	Type        string `json:"type"`
	File        string `json:"file"`
	AuthorRowId string `json:"author_row_id"`
	ParentId    any    `json:"parent_id"`
	Offset      uint64 `json:"offset"`
	Limit       uint64 `json:"limit"`
}

type ReadMessage struct {
	RoomId string `json:"room_id"`
	RowId  string `json:"row_id"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type UpdateMessage struct {
	Id      string `json:"id"`
	Content string `json:"content"`
	File    string `json:"file"`
	Type    string `json:"type"`
}

type RoomsList struct {
	RowId  string `json:"row_id"`
	Type   string `json:"type"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type PresenceConnected struct {
	RowId     string `json:"row_id"`
	ProjectId string `json:"project_id"`
}

type PresencePing struct {
	RowId     string `json:"row_id"`
	ProjectId string `json:"project_id"`
}
type PresenceGet struct {
	RowId string `json:"row_id"`
}
