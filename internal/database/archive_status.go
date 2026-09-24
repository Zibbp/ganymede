package database

import (
	"context"
	"database/sql"
	"fmt"
)

// migrateArchiveStatus runs before Ent adds defaults so legacy state is never
// temporarily mistaken for completion. Backfill and obsolete-column removal
// commit together; a failed migration is safe to retry on either process.
func migrateArchiveStatus(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var legacy bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'vods' AND column_name = 'processing'
	)`).Scan(&legacy); err != nil {
		return err
	}
	if !legacy {
		return nil
	}
	statements := []string{
		`ALTER TABLE vods ADD COLUMN status varchar NOT NULL DEFAULT 'completed'`,
		`UPDATE vods v SET status = CASE
			WHEN NOT v.processing THEN 'completed'
			WHEN q.id IS NULL THEN 'failed'
			WHEN 'failed' IN (q.task_vod_create_folder, q.task_vod_save_info, q.task_vod_download_thumbnail,
				q.task_video_download, q.task_video_convert, q.task_video_move) THEN 'failed'
			WHEN q.task_video_download = 'success' AND q.task_video_convert = 'success' AND q.task_video_move = 'success' THEN
				CASE WHEN NOT q.archive_chat
					OR 'failed' IN (q.task_chat_download, q.task_chat_move)
					OR (q.live_archive AND q.task_chat_convert = 'failed')
					OR (q.render_chat AND q.task_chat_render = 'failed')
					OR (q.task_chat_download = 'success' AND q.task_chat_move = 'success'
						AND (NOT q.live_archive OR q.task_chat_convert = 'success')
						AND (NOT q.render_chat OR q.task_chat_render = 'success'))
				THEN 'completed' ELSE 'finalizing' END
			WHEN q.task_video_download = 'running' THEN 'running'
			WHEN q.task_video_download = 'success' THEN 'finalizing'
			ELSE 'queued'
		END
		FROM (SELECT v0.id AS vod_id, q0.* FROM vods v0 LEFT JOIN queues q0 ON q0.vod_queue = v0.id) q
		WHERE v.id = q.vod_id`,
		`ALTER TABLE vods DROP COLUMN processing`,
		`ALTER TABLE queues DROP COLUMN processing, DROP COLUMN video_processing, DROP COLUMN chat_processing`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate archive status: %w", err)
		}
	}
	return tx.Commit()
}
