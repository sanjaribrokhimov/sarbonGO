package trips

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPendingDriver = "pending_driver"
	StatusAssigned      = "assigned"
	StatusLoading       = "loading"
	StatusEnRoute       = "en_route"
	StatusUnloading     = "unloading"
	StatusCompleted     = "completed"
	StatusCancelled     = "cancelled"
)

type Trip struct {
	ID        uuid.UUID
	CargoID   uuid.UUID
	OfferID   uuid.UUID
	DriverID  *uuid.UUID
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
