package admins

import "github.com/google/uuid"

type Admin struct {
	ID       uuid.UUID `json:"id"`
	Login    string    `json:"login"`
	Password string    `json:"-"` // bcrypt hash
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Type     string    `json:"type"`
}

