package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
)

// ApiKey holds the schema definition for an admin-managed API key used to
// authenticate external scripts against the Ganymede HTTP API.
type ApiKey struct {
	ent.Schema
}

// Fields of the ApiKey.
func (ApiKey) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").Unique().NotEmpty(),
		field.String("description").Optional(),
		// prefix is the publicly visible identifier portion of the token
		// (e.g. "gym_abc123def456"). Indexed for O(log n) lookup on every
		// authenticated request.
		field.String("prefix").Unique().Immutable().NotEmpty(),
		// hashed_secret is bcrypt(secret_half_of_token). Sensitive() prevents
		// it from being printed via %v / zerolog struct logging, but does NOT
		// stop JSON marshalling — handlers must scrub it via DTOs.
		field.String("hashed_secret").Sensitive().Immutable().NotEmpty(),
		field.Enum("scope").GoType(utils.ApiKeyScope("")),
		field.Time("last_used_at").Optional().Nillable(),
		// revoked_at is the soft-delete marker. List queries filter where
		// revoked_at IS NULL.
		field.Time("revoked_at").Optional().Nillable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the ApiKey.
func (ApiKey) Edges() []ent.Edge {
	return nil
}
