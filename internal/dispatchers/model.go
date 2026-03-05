package dispatchers

import "time"

// Mirrors DB columns from tables:
// - freelance_dispatchers
// - deleted_freelance_dispatchers
type Dispatcher struct {
	ID string `json:"id"`

	Name     *string `json:"name"`
	Phone    string  `json:"phone"`
	Password string  `json:"password"` // stored as bcrypt hash

	PassportSeries *string `json:"passport_series"`
	PassportNumber *string `json:"passport_number"`
	PINFL          *string `json:"pinfl"`

	CargoID  *string `json:"cargo_id"`
	DriverID *string `json:"driver_id"`

	Rating     *float64 `json:"rating"`
	WorkStatus *string  `json:"work_status"`
	Status     *string  `json:"status"`

	Photo     *string `json:"photo,omitempty"`      // ссылка/путь (устаревшее), для загруженного фото см. has_photo
	HasPhoto  bool    `json:"has_photo"`             // true если загружено фото в БД (получить через GET /profile/photo)

	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastOnlineAt *time.Time `json:"last_online_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}
