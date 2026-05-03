package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
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
		// hashed_secret is bcrypt(secret_half_of_token). Sensitive()
		// does two things: prevents printing via %v / zerolog struct
		// logging (the field renders as <sensitive> in String()), and
		// generates a `json:"-"` tag on the struct field so direct
		// JSON marshalling of *ent.ApiKey drops it. apiKeyDTO is the
		// authoritative wire shape — the json tag is belt-and-
		// suspenders for any future code that marshals the entity
		// directly (debug dumps, logging libraries, etc.).
		field.String("hashed_secret").Sensitive().Immutable().NotEmpty(),
		// scopes is the list of granted permissions, each formatted as
		// "<resource>:<tier>" (utils.ApiKeyScope). Stored as a JSON column;
		// validated by the service layer before persistence. Replaces the
		// previous single-tier `scope` ENUM column — see commit message.
		field.JSON("scopes", []string{}).Default([]string{}),
		// created_by_id is the FK column for the created_by edge below.
		// Optional/nullable so rows that predate the audit edge remain
		// valid; new keys minted via /admin/api-keys always set it to
		// the session user's id.
		field.UUID("created_by_id", uuid.UUID{}).Optional().Nillable(),
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
	return []ent.Edge{
		// created_by records which admin user minted this key. Optional
		// (Unique).Field("created_by_id") so ent stores the FK on the
		// api_keys table rather than a join table; nullable so existing
		// rows pre-migration remain valid.
		//
		// ON DELETE SET NULL so deleting the admin who minted a key
		// doesn't fail with a foreign-key violation: the key stays
		// (it's still valid auth material) and the audit reference
		// becomes null. Without this, an admin trying to remove a
		// colleague's user account would get a 500 from the FK
		// constraint as soon as that colleague had ever minted a key.
		edge.To("created_by", User.Type).
			Unique().
			Field("created_by_id").
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
