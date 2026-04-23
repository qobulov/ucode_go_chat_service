package postgres

import (
	"context"
	"time"
	"ucode/ucode_go_chat_service/config"
	"ucode/ucode_go_chat_service/models"
	"ucode/ucode_go_chat_service/pkg/db"
	"ucode/ucode_go_chat_service/pkg/logger"
)

var (
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastSeenAt time.Time
	ReadAt     time.Time
)

type postgresRepo struct {
	Db  *db.Postgres
	Log *logger.Logger
	Cfg config.Config
}

func New(db *db.Postgres, log *logger.Logger, cfg config.Config) PostgresI {
	return &postgresRepo{
		Db:  db,
		Log: log,
		Cfg: cfg,
	}
}

type PostgresI interface {
	RoomCreate(ctx context.Context, req *models.CreateRoom) (*models.Room, error)
	RoomGetSingle(ctx context.Context, req *models.GetSingleRoom) (*models.Room, error)
	RoomGetList(ctx context.Context, req *models.GetListRoomReq) (*models.GetListRoomResp, error)
	RoomExists(ctx context.Context, req *models.ExistsRoom) (string, error)
	UnreadCountInRoom(ctx context.Context, req *models.UnreadCountReq) (*models.UnreadCountResp, error)
	RoomIdByItemId(ctx context.Context, req *models.GetRoomIdByItemIdReq) (*models.GetRoomIdByItemIdResp, error)
	RoomDelete(ctx context.Context, id string) error

	RoomMemberCreate(ctx context.Context, req *models.CreateRoomMember) (*models.RoomMember, error)
	RoomMembersByRoomId(ctx context.Context, roomId string) ([]*models.RoomMember, error)
	UpdateLastReadAt(ctx context.Context, req *models.UpdateLastReadAtReq) error

	MessageCreate(ctx context.Context, req *models.CreateMessage) (*models.Message, error)
	MessageGetList(ctx context.Context, req *models.GetListMessageReq) (*models.GetListMessageResp, error)
	MessageUpdate(ctx context.Context, req *models.UpdateMessage) (*models.UpdateMessage, error)
	MessageMarkRead(ctx context.Context, req *models.MarkReadMessage) (*models.MarkReadMessageResp, error)

	PresenceUpsert(ctx context.Context, req *models.UpsertPresence) error
	PresenceHeartbeat(ctx context.Context, req *models.HeartbeatPresence) error
	PresenceGet(ctx context.Context, req *models.GetPresence) (*models.Presence, error)
	PresenceSweepOffline(ctx context.Context, cutoff time.Time) ([]string, error)
}
