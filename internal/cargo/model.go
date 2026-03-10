package cargo

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CargoStatus values.
const (
	StatusCreated    = "created"
	StatusSearching  = "searching"
	StatusAssigned   = "assigned"
	StatusInTransit  = "in_transit"
	StatusDelivered  = "delivered"
	StatusCancelled  = "cancelled"
)

// Documents is the JSON object for cargo.documents (TIR, T1, CMR, etc.).
type Documents struct {
	TIR      *bool `json:"TIR,omitempty"`
	T1       *bool `json:"T1,omitempty"`
	CMR      *bool `json:"CMR,omitempty"`
	Medbook  *bool `json:"Medbook,omitempty"`
	GLONASS  *bool `json:"GLONASS,omitempty"`
	Seal     *bool `json:"Seal,omitempty"`
	Permit   *bool `json:"Permit,omitempty"`
}

// Cargo model (table cargo).
type Cargo struct {
	ID            uuid.UUID
	Weight        float64
	Volume        float64
	ReadyEnabled  bool
	ReadyAt       *time.Time
	LoadComment   *string
	TruckType     string
	TempMin       *float64
	TempMax       *float64
	ADREnabled    bool
	ADRClass      *string
	LoadingTypes  []string
	Requirements  []string
	ShipmentType  *string
	BeltsCount    *int
	Documents     *Documents
	ContactName   *string
	ContactPhone  *string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
	// Кто создал: admin, dispatcher или company (admins, freelance_dispatchers или companies)
	CreatedByType *string   // "admin" | "dispatcher" | "company"
	CreatedByID   *uuid.UUID
	// От какой компании груз (опционально; при created_by_type=company совпадает с created_by_id)
	CompanyID     *uuid.UUID
}

// RoutePoint model (table route_points).
type RoutePoint struct {
	ID           uuid.UUID
	CargoID      uuid.UUID
	Type         string // load, unload, customs, transit
	CityCode     string // код города (TAS, SAM, DXB и т.д.) — из справочника cities
	RegionCode   string // код региона/области — из справочника regions
	Address      string // адрес (улица, дом)
	Orientir     string // ориентир (что написать для водителя)
	Lat          float64
	Lng          float64
	Comment      *string
	PointOrder   int
	IsMainLoad   bool
	IsMainUnload bool
}

// Payment model (table payments).
type Payment struct {
	ID                uuid.UUID
	CargoID           uuid.UUID
	IsNegotiable      bool
	PriceRequest      bool
	TotalAmount       *float64
	TotalCurrency     *string
	WithPrepayment    bool
	WithoutPrepayment bool
	PrepaymentAmount  *float64
	PrepaymentCurrency *string
	PrepaymentType    *string
	RemainingAmount   *float64
	RemainingCurrency *string
	RemainingType     *string
}

// Offer model (table offers).
type Offer struct {
	ID        uuid.UUID
	CargoID   uuid.UUID
	CarrierID uuid.UUID
	Price     float64
	Currency  string
	Comment   *string
	Status    string // pending, accepted, rejected
	CreatedAt time.Time
}

// DocumentsToJSON returns JSON bytes for DB insert/update.
func DocumentsToJSON(d *Documents) ([]byte, error) {
	if d == nil {
		return nil, nil
	}
	return json.Marshal(d)
}

// DocumentsFromJSON parses jsonb from DB.
func DocumentsFromJSON(b []byte) (*Documents, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var d Documents
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
