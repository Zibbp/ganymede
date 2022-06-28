package user

import (
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
)

type User struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	Password  string     `json:"password"`
	Role      utils.Role `json:"role"`
	Webhook   string     `json:"webhook"`
	UpdatedAt string     `json:"updated_at"`
	CreatedAt string     `json:"created_at"`
}
