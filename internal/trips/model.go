package trips

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPendingDriver = "PENDING_DRIVER"
	StatusAssigned      = "ASSIGNED"
	StatusLoading       = "LOADING"
	StatusEnRoute       = "EN_ROUTE"
	StatusUnloading     = "UNLOADING"
	StatusCompleted     = "COMPLETED"
	StatusCancelled     = "CANCELLED"
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
