package postgres

import (
	"context"
	"database/sql"
	"time"
	"ucode/ucode_go_chat_service/models"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *postgresRepo) MessageCreate(ctx context.Context, req *models.CreateMessage) (*models.Message, error) {
	res := &models.Message{}
	id := uuid.NewString()

	sqlStr, args, err := r.Db.Builder.
		Insert("messages").
		Columns("id", "room_id", "message", `"type"`, "file", `"from"`, "author_row_id", "read_at", "parent_id").
		Values(id, req.RoomId, req.Message, req.Type, req.File, req.From, req.AuthorRowId, nil, req.ParentId).
		Suffix(`RETURNING id, room_id, message, "type", file, "from", author_row_id, read_at, parent_id, created_at, updated_at`).
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageCreate: build sql")
	}

	var (
		readAt   sql.NullTime
		parentId sql.NullString
	)

	err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(
		&res.Id,
		&res.RoomId,
		&res.Message,
		&res.Type,
		&res.File,
		&res.From,
		&res.AuthorRowId,
		&readAt,
		&parentId,
		&CreatedAt,
		&UpdatedAt,
	)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageCreate: query run")
	}

	res.ParentId = parentId.String
	res.CreatedAt = CreatedAt.Format(time.RFC1123)
	res.UpdatedAt = UpdatedAt.Format(time.RFC1123)
	if readAt.Valid {
		res.ReadAt = readAt.Time.Format(time.RFC1123)
	}

	return res, nil
}

func (r *postgresRepo) MessageUpdate(ctx context.Context, req *models.UpdateMessage) (*models.UpdateMessage, error) {
	sqlStr, args, err := r.Db.Builder.
		Update("messages").
		SetMap(map[string]any{
			"message":    req.Content,
			"file":       req.File,
			"type":       req.Type,
			"updated_at": sq.Expr("CURRENT_TIMESTAMP"),
		}).
		Where(sq.Eq{"id": req.Id}).
		Suffix(`RETURNING id, message AS content, file, type`).
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageUpdate: build sql")
	}

	updatedMsg := &models.UpdateMessage{}

	err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(
		&updatedMsg.Id,
		&updatedMsg.Content,
		&updatedMsg.File,
		&updatedMsg.Type,
	)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageUpdate: query run")
	}

	return updatedMsg, nil
}

func (r *postgresRepo) MessageGetList(ctx context.Context, req *models.GetListMessageReq) (*models.GetListMessageResp, error) {
	res := &models.GetListMessageResp{}

	inner := r.Db.Builder.
		Select(
			"id",
			"room_id",
			"message",
			`"type"`,
			"file",
			`"from"`,
			"author_row_id",
			"created_at",
			"updated_at",
			"read_at",
			"parent_id",
			"COUNT(*) OVER() AS total_count",
		).
		From("messages").
		Where(sq.Eq{"room_id": req.RoomId}).
		OrderBy("created_at DESC", "id DESC").
		Limit(req.Limit).
		Offset(req.Offset)

	outer := r.Db.Builder.
		Select(
			"id",
			"room_id",
			"message",
			`"type"`,
			"file",
			`"from"`,
			"author_row_id",
			"created_at",
			"updated_at",
			"read_at",
			"parent_id",
			"total_count",
		).
		FromSelect(inner, "picked").
		OrderBy("created_at ASC")

	sqlStr, args, err := outer.ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageGetList: build sql")
	}

	rows, err := r.Db.Pg.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageGetList: query run")
	}
	defer rows.Close()

	first := true
	for rows.Next() {
		msg := &models.Message{}
		var (
			totalCnt uint64
			readAt   sql.NullTime
			parentId sql.NullString
		)

		if err = rows.Scan(
			&msg.Id,
			&msg.RoomId,
			&msg.Message,
			&msg.Type,
			&msg.File,
			&msg.From,
			&msg.AuthorRowId,
			&CreatedAt,
			&UpdatedAt,
			&readAt,
			&parentId,
			&totalCnt,
		); err != nil {
			return nil, HandleDatabaseError(err, r.Log, "MessageGetList: row scan")
		}

		if first {
			res.Count = totalCnt
			first = false
		}

		msg.ParentId = parentId.String
		msg.CreatedAt = CreatedAt.Format(time.RFC1123)
		msg.UpdatedAt = UpdatedAt.Format(time.RFC1123)
		if readAt.Valid {
			msg.ReadAt = readAt.Time.Format(time.RFC1123)
		}

		res.Messages = append(res.Messages, msg)
	}

	return res, nil
}

func (r *postgresRepo) MessageMarkRead(ctx context.Context, req *models.MarkReadMessage) (*models.MarkReadMessageResp, error) {
	resp := &models.MarkReadMessageResp{
		RoomId: req.RoomId,
	}

	if req.ReadAt.IsZero() {
		req.ReadAt = time.Now().UTC()
	}
	resp.ReadAt = req.ReadAt.Format(time.RFC1123)

	sqlStr, args, err := r.Db.Builder.
		Update("messages").
		Set("read_at", req.ReadAt).
		Where(sq.Eq{"room_id": req.RoomId}).
		Where(sq.NotEq{"author_row_id": req.ReaderRowId}).
		Where("read_at IS NULL").
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageMarkRead: build sql")
	}

	ct, err := r.Db.Pg.Exec(ctx, sqlStr, args...)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "MessageMarkRead: exec")
	}

	if ct.RowsAffected() > 0 {
		resp.Updated = true
	}

	return resp, nil
}
