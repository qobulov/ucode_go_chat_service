package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"ucode/ucode_go_chat_service/models"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *postgresRepo) RoomCreate(ctx context.Context, req *models.CreateRoom) (*models.Room, error) {
	var (
		res        = &models.Room{}
		id         = uuid.NewString()
		itemId     sql.NullString
		attributes []byte
	)

	if len(req.Attributes) == 0 {
		attributes = []byte("{}")
	} else {
		attributes = req.Attributes
	}

	sqlStr, args, err := r.Db.Builder.
		Insert("rooms").
		Columns("id", "name", "type", "project_id", "item_id", "attributes").
		Values(id, req.Name, req.Type, req.ProjectId, req.ItemId, attributes).
		Suffix("RETURNING id, name, type, project_id, item_id, attributes, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomCreate: build sql")
	}

	err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(
		&res.Id,
		&res.Name,
		&res.Type,
		&res.ProjectId,
		&itemId,
		&res.Attributes,
		&CreatedAt,
		&UpdatedAt,
	)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomCreate: query run")
	}

	res.ItemId = itemId.String
	res.CreatedAt = CreatedAt.Format(time.RFC1123)
	res.UpdatedAt = UpdatedAt.Format(time.RFC1123)

	return res, nil
}

func (r *postgresRepo) RoomGetSingle(ctx context.Context, req *models.GetSingleRoom) (*models.Room, error) {
	room := &models.Room{}

	sqlStr, args, err := r.Db.Builder.
		Select("r.id", "r.name", "r.type", "r.project_id", "r.item_id", "r.attributes", "r.created_at", "r.updated_at").
		From("rooms r").
		Where(sq.Eq{"r.id": req.Id}).
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomGetById: build sql")
	}

	err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(
		&room.Id,
		&room.Name,
		&room.Type,
		&room.ProjectId,
		&room.ItemId,
		&room.Attributes,
		&CreatedAt,
		&UpdatedAt,
	)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomGetById: query run")
	}

	room.CreatedAt = CreatedAt.Format(time.RFC1123)
	room.UpdatedAt = UpdatedAt.Format(time.RFC1123)

	return room, nil
}

func (r *postgresRepo) RoomGetList(ctx context.Context, req *models.GetListRoomReq) (*models.GetListRoomResp, error) {
	res := &models.GetListRoomResp{Rooms: make([]*models.Room, 0)}

	builder := r.Db.Builder.
		Select(
			"r.id",
			"r.name",
			"r.type",
			"r.project_id",
			"r.item_id",
			"r.attributes",
			"rm.to_name",
			"rm.to_row_id",
			"r.created_at",
			"r.updated_at",
			"lm.message AS last_message",
			"lm.type AS last_message_type",
			"lm.file AS last_message_file",
			"lm.from",
			"lm.created_at AS last_message_created_at",
			"up.status AS user_presence_status",
			"up.last_seen_at AS user_presence_last_seen_at",
			"(SELECT COUNT(*) FROM room_members rm2 WHERE rm2.room_id = r.id) AS member_count",
			"COUNT(*) OVER()",
		).
		From("rooms r").
		Join("room_members rm ON rm.room_id = r.id").
		LeftJoin(`
            (
                SELECT DISTINCT ON (m.room_id)
                    m.room_id,
                    m.message,
                    m.type,
                    m.file,
                    m.from,
                    m.created_at
                FROM messages m
                ORDER BY m.room_id, m.created_at DESC
            ) AS lm ON lm.room_id = r.id
        `).
		LeftJoin(`
            (
                SELECT DISTINCT ON (row_id)
                    row_id,
                    status,
                    last_seen_at
                FROM user_presence
                ORDER BY row_id, last_seen_at DESC
            ) AS up ON up.row_id = rm.to_row_id
        `).
		Where(sq.Eq{"rm.row_id": req.RowId}).
		OrderBy("r.updated_at DESC").
		Limit(req.Limit).
		Offset(req.Offset)

	if req.Type != "" {
		builder = builder.Where(sq.Eq{"r.type": req.Type})
	}
	if req.ProjectId != "" {
		builder = builder.Where(sq.Eq{"r.project_id": req.ProjectId})
	}
	if req.Search != "" {
		builder = builder.Where(sq.Or{
			sq.ILike{"rm.to_name": "%" + req.Search + "%"},
			sq.ILike{"r.name": "%" + req.Search + "%"},
		})
	}

	sqlStr, args, err := builder.ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomGetList: build sql")
	}

	rows, err := r.Db.Pg.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomGetList: query run")
	}
	defer rows.Close()

	first := true
	for rows.Next() {
		room := &models.Room{}
		var (
			totalCnt                                                                            uint64
			lastMsg, lastMsgType, lastMsgFile, lastMsgFrom, itemId, toRowId, userPresenceStatus sql.NullString
			lastMessageCreatedAt, userPresenceLastSeen                                          sql.NullTime
			CreatedAt, UpdatedAt                                                                time.Time
		)

		if err = rows.Scan(
			&room.Id,
			&room.Name,
			&room.Type,
			&room.ProjectId,
			&itemId,
			&room.Attributes,
			&room.ToName,
			&toRowId,
			&CreatedAt,
			&UpdatedAt,
			&lastMsg,
			&lastMsgType,
			&lastMsgFile,
			&lastMsgFrom,
			&lastMessageCreatedAt,
			&userPresenceStatus,
			&userPresenceLastSeen,
			&room.MemberCount,
			&totalCnt,
		); err != nil {
			return nil, HandleDatabaseError(err, r.Log, "RoomGetList: row scan")
		}

		if first {
			res.Count = totalCnt
			first = false
		}

		room.CreatedAt = CreatedAt.Format(time.RFC1123)
		room.UpdatedAt = UpdatedAt.Format(time.RFC1123)
		room.LastMessage = lastMsg.String
		room.LastMessageFrom = lastMsgFrom.String
		room.LastMessageType = lastMsgType.String
		room.LastMessageFile = lastMsgFile.String
		if lastMessageCreatedAt.Valid {
			room.LastMessageCreatedAt = lastMessageCreatedAt.Time.Format(time.RFC1123)
		}
		room.ItemId = itemId.String
		room.ToRowId = toRowId.String

		room.UserPresenceStatus = userPresenceStatus.String
		if userPresenceLastSeen.Valid {
			room.UserPresenceLastSeenAt = userPresenceLastSeen.Time.Format(time.RFC1123)
		}

		unreadCount, _ := r.UnreadCountInRoom(ctx, &models.UnreadCountReq{
			RoomId: room.Id,
			RowId:  req.RowId,
		})

		room.UnreadMessageCount = unreadCount.Count

		res.Rooms = append(res.Rooms, room)
	}

	return res, nil
}

func (r *postgresRepo) RoomExists(ctx context.Context, req *models.ExistsRoom) (string, error) {
	var (
		roomID string
		sqlStr string
		args   []any
		err    error
	)

	if req.Type == "single" {
		sqlStr, args, err = r.Db.Builder.
			Select("r.id").
			From("rooms r").
			Join("room_members rm1 ON rm1.room_id = r.id").
			Join("room_members rm2 ON rm2.room_id = r.id").
			Where(sq.And{
				sq.Eq{"r.project_id": req.ProjectId},
				sq.Eq{"r.type": req.Type},
				sq.Eq{"rm1.row_id": req.RowId},
				sq.Eq{"rm2.row_id": req.ToRowId},
			}).
			Limit(1).
			ToSql()
		if err != nil {
			return "", HandleDatabaseError(err, r.Log, "RoomExists(single): build sql")
		}

		err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(&roomID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", nil
			}
			return "", HandleDatabaseError(err, r.Log, "RoomExists(single): query run")
		}
	} else {
		sqlStr, args, err = r.Db.Builder.
			Select("r.id").
			From("rooms r").
			Where(sq.And{
				sq.Eq{"r.project_id": req.ProjectId},
				sq.Eq{"r.type": req.Type},
				sq.Eq{"r.item_id": req.ItemId},
			}).
			Limit(1).
			ToSql()
		if err != nil {
			return "", HandleDatabaseError(err, r.Log, "RoomExists(group): build sql")
		}

		err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(&roomID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", nil
			}
			return "", HandleDatabaseError(err, r.Log, "RoomExists(group): query run")
		}
	}

	return roomID, nil
}

func (r *postgresRepo) UnreadCountInRoom(ctx context.Context, req *models.UnreadCountReq) (*models.UnreadCountResp, error) {
	resp := &models.UnreadCountResp{RoomId: req.RoomId, RowId: req.RowId}

	sqlStr, args, err := r.Db.Builder.
		Select("COUNT(*)").
		From("messages m").
		Where(sq.Eq{"m.room_id": req.RoomId}).
		Where(sq.NotEq{"m.author_row_id": req.RowId}).
		Where(sq.Expr(`
			m.created_at > COALESCE(
				(SELECT rm.last_read_at
				 FROM room_members rm
				 WHERE rm.room_id = ? AND rm.row_id = ?
				 LIMIT 1),
				TO_TIMESTAMP(0)
			)
		`, req.RoomId, req.RowId)).
		ToSql()
	if err != nil {
		return &models.UnreadCountResp{}, HandleDatabaseError(err, r.Log, "UnreadCountInRoom: build sql")
	}

	var count int64
	if err := r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(&count); err != nil {
		return &models.UnreadCountResp{}, HandleDatabaseError(err, r.Log, "UnreadCountInRoom: scan")
	}
	resp.Count = count
	return resp, nil
}

func (r *postgresRepo) RoomIdByItemId(ctx context.Context, req *models.GetRoomIdByItemIdReq) (*models.GetRoomIdByItemIdResp, error) {
	var (
		roomId sql.NullString
	)

	sqlStr, args, err := r.Db.Builder.
		Select("id").
		From("rooms").
		Where(sq.Eq{
			"item_id":    req.ItemId,
			"project_id": req.ProjectId,
		}).
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomIdByItemId: build sql")
	}

	err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(&roomId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.GetRoomIdByItemIdResp{
				RoomId: "",
			}, nil
		}
		return nil, HandleDatabaseError(err, r.Log, "RoomIdByItemId: query run")
	}

	return &models.GetRoomIdByItemIdResp{
		RoomId: roomId.String,
	}, nil
}
