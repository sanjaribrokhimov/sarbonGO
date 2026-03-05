package appusers

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           string     `json:"id"`
	Phone        string     `json:"phone"`
	PasswordHash string     `json:"-"`
	FirstName    *string    `json:"first_name"`
	LastName     *string    `json:"last_name"`
	CompanyID    *uuid.UUID `json:"company_id"`
	Role         *string    `json:"role"` // owner, etc.
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
