package appusers

import "time"

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Phone        *string    `json:"phone"`
	PasswordHash string     `json:"-"`
	FirstName    *string    `json:"first_name"`
	LastName     *string   `json:"last_name"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
