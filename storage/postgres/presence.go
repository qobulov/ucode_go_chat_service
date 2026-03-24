package postgres

import (
	"context"
	"time"
	"ucode/ucode_go_chat_service/models"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func (r *postgresRepo) PresenceUpsert(ctx context.Context, req *models.UpsertPresence) error {
	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}
	if req.Status == "" {
		req.Status = "online"
	}

	sqlStr, args, err := r.Db.Builder.
		Insert("user_presence").
		Columns("row_id", "project_id", "status", "last_seen_at", "updated_at").
		Values(req.RowId, req.ProjectId, req.Status, req.Now, req.Now).
		Suffix(`
			ON CONFLICT (row_id, project_id) DO UPDATE
			SET status = EXCLUDED.status,
			    last_seen_at = EXCLUDED.last_seen_at,
			    updated_at = EXCLUDED.updated_at
		`).
		ToSql()
	if err != nil {
		return HandleDatabaseError(err, r.Log, "PresenceUpsert: build sql")
	}

	if _, err := r.Db.Pg.Exec(ctx, sqlStr, args...); err != nil {
		return HandleDatabaseError(err, r.Log, "PresenceUpsert: exec")
	}

	return nil
}

func (r *postgresRepo) PresenceHeartbeat(ctx context.Context, req *models.HeartbeatPresence) error {
	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}

	sqlStr, args, err := r.Db.Builder.
		Insert("user_presence").
		Columns("row_id", "project_id", "status", "last_seen_at", "updated_at").
		Values(req.RowId, req.ProjectId, "online", req.Now, req.Now).
		Suffix(`
			ON CONFLICT (row_id, project_id) DO UPDATE
			SET status = 'online',
			    last_seen_at = EXCLUDED.last_seen_at,
			    updated_at = EXCLUDED.updated_at
		`).
		ToSql()
	if err != nil {
		return HandleDatabaseError(err, r.Log, "PresenceHeartbeat: build sql")
	}

	if _, err := r.Db.Pg.Exec(ctx, sqlStr, args...); err != nil {
		return HandleDatabaseError(err, r.Log, "PresenceHeartbeat: exec")
	}

	return nil
}

func (r *postgresRepo) PresenceGet(ctx context.Context, req *models.GetPresence) (*models.Presence, error) {
	sqlStr, args, err := r.Db.Builder.
		Select("row_id", "status", "last_seen_at").
		From("user_presence").
		Where(sq.Eq{"row_id": req.RowId}).
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "PresenceGet: build sql")
	}

	var (
		res = &models.Presence{}
	)
	if err := r.Db.Pg.QueryRow(ctx, sqlStr, args...).Scan(
		&res.RowId,
		&res.Status,
		&LastSeenAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, HandleDatabaseError(err, r.Log, "PresenceGet: query row")
	}

	res.LastSeenAt = LastSeenAt.Format(time.RFC1123)

	return res, nil
}
func (r *postgresRepo) PresenceSweepOffline(ctx context.Context, cutoff time.Time) ([]string, error) {
	sqlStr, args, err := r.Db.Builder.
		Update("user_presence").
		Set("status", "offline").
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Lt{"last_seen_at": cutoff}).
		Where(sq.NotEq{"status": "offline"}).
		Suffix("RETURNING row_id").
		ToSql()
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "PresenceSweepOfflineIDs: build sql")
	}

	rows, err := r.Db.Pg.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, HandleDatabaseError(err, r.Log, "PresenceSweepOfflineIDs: exec")
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, HandleDatabaseError(err, r.Log, "PresenceSweepOfflineIDs: scan")
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, HandleDatabaseError(err, r.Log, "PresenceSweepOfflineIDs: rows err")
	}

	return ids, nil
}
