package tasks_shared

import (
	"context"
	"database/sql"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type contextKey string

const StoreKey contextKey = "store"
const PlatformTwitchKey contextKey = "platform_twitch"
const LiveServiceKey contextKey = "live_service"
const NotificationServiceKey contextKey = "notification_service"
const EnqueuerKey contextKey = "enqueuer"

type Enqueuer interface {
	Insert(context.Context, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error)
	InsertTx(context.Context, *sql.Tx, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error)
}
