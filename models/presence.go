package models

import "time"

type Presence struct {
	RowId      string `json:"row_id"`
	Status     string `json:"status"`
	LastSeenAt string `json:"last_seen_at"`
}

type UpsertPresence struct {
	RowId     string
	ProjectId string
	Status    string
	Now       time.Time
}

type HeartbeatPresence struct {
	RowId     string
	Now       time.Time
	ProjectId string
}

type GetPresence struct {
	RowId string
}
