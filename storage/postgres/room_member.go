package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"ucode/ucode_go_chat_service/models"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cast"
)

func (r *postgresRepo) RoomMemberCreate(ctx context.Context, req *models.CreateRoomMember) (*models.RoomMember, error) {
	var (
		res        = &models.RoomMember{}
		id         = uuid.NewString()
		attributes []byte
	)

	if len(req.Attributes) == 0 {
		attributes = []byte("{}")
	} else {
		attributes = req.Attributes
	}

	sqlStr, args, err := r.Db.Builder.
		Insert("room_members").
		Columns("id", "room_id", "row_id", "to_name", "to_row_id", "attributes").
		Values(id, req.RoomId, req.RowId, req.ToName, req.ToRowId, attributes).
		Suffix(`
			ON CONFLICT (room_id, row_id) DO NOTHING 
			RETURNING id, room_id, row_id, to_name, to_row_id, attributes, created_at, updated_at
		`).
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomMemberCreate: build sql")
	}

	err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(
		&res.Id,
		&res.RoomId,
		&res.RowId,
		&res.ToName,
		&res.ToRowId,
		&res.Attributes,
		&CreatedAt,
		&UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, HandleDatabaseError(err, r.Log, "RoomMemberCreate: query run")
	}

	res.CreatedAt = CreatedAt.Format(time.RFC1123)
	res.UpdatedAt = UpdatedAt.Format(time.RFC1123)

	return res, nil
}

func (r *postgresRepo) RoomMembersByRoomId(ctx context.Context, roomId string) ([]*models.RoomMember, error) {
	sqlStr, args, err := r.Db.Builder.
		Select("id", "room_id", "row_id", "to_name", "to_row_id", "attributes", "created_at", "updated_at").
		From("room_members").
		Where(sq.Eq{"room_id": roomId}).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.Db.Pg.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*models.RoomMember
	for rows.Next() {
		rm := &models.RoomMember{}
		var CreatedAt, UpdatedAt time.Time
		if err := rows.Scan(
			&rm.Id,
			&rm.RoomId,
			&rm.RowId,
			&rm.ToName,
			&rm.ToRowId,
			&rm.Attributes,
			&CreatedAt,
			&UpdatedAt,
		); err != nil {
			return nil, err
		}
		rm.CreatedAt = CreatedAt.Format(time.RFC1123)
		rm.UpdatedAt = UpdatedAt.Format(time.RFC1123)
		res = append(res, rm)
	}

	return res, nil
}

func (r *postgresRepo) RoomMemberUpdate(ctx context.Context, req *models.UpdateRoomMember) (*models.RoomMember, error) {
	setMap := map[string]any{
		"updated_at": sq.Expr("CURRENT_TIMESTAMP"),
	}
	if req.ToName != "" {
		setMap["to_name"] = req.ToName
	}
	if toRowId := cast.ToString(req.ToRowId); toRowId != "" {
		setMap["to_row_id"] = toRowId
	}
	if len(req.Attributes) > 0 {
		setMap["attributes"] = req.Attributes
	}
	if len(setMap) == 1 {
		return nil, fmt.Errorf("no fields to update")
	}

	sqlStr, args, err := r.Db.Builder.
		Update("room_members").
		SetMap(setMap).
		Where(sq.Eq{"room_id": req.RoomId, "row_id": req.RowId}).
		Suffix("RETURNING id, room_id, row_id, to_name, to_row_id, attributes, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomMemberUpdate: build sql")
	}

	res := &models.RoomMember{}
	err = r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(
		&res.Id,
		&res.RoomId,
		&res.RowId,
		&res.ToName,
		&res.ToRowId,
		&res.Attributes,
		&CreatedAt,
		&UpdatedAt,
	)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "RoomMemberUpdate: query run")
	}

	res.CreatedAt = CreatedAt.Format(time.RFC1123)
	res.UpdatedAt = UpdatedAt.Format(time.RFC1123)
	return res, nil
}

func (r *postgresRepo) RoomMemberDelete(ctx context.Context, roomId, rowId string) error {
	sqlStr, args, err := r.Db.Builder.
		Delete("room_members").
		Where(sq.Eq{"room_id": roomId, "row_id": rowId}).
		ToSql()
	if err != nil {
		return HandleDatabaseError(err, r.Log, "RoomMemberDelete: build sql")
	}
	tag, err := r.Db.Pg.Exec(ctx, sqlStr, args...)
	if err != nil {
		return HandleDatabaseError(err, r.Log, "RoomMemberDelete: exec")
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("room member not found")
	}
	return nil
}

func (r *postgresRepo) UpdateLastReadAt(ctx context.Context, req *models.UpdateLastReadAtReq) error {
	sqlStr, args, err := r.Db.Builder.
		Update("room_members").
		Set("last_read_at", "NOW()").
		Where(sq.Eq{
			"room_id": req.RoomId,
			"row_id":  req.RowId,
		}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.Db.Pg.Exec(ctx, sqlStr, args...)
	return err
}
