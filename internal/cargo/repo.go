package cargo

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

// ListFilter for GET /api/cargo.
type ListFilter struct {
	Status      []string // status=created,searching
	WeightMin   *float64
	WeightMax   *float64
	TruckType   string
	CreatedFrom string // YYYY-MM-DD
	CreatedTo   string
	WithOffers  *bool   // only cargo that have at least one offer
	Page        int
	Limit       int
	Sort        string // "created_at:desc" or "created_at:asc"
}

// ListResult for paginated list.
type ListResult struct {
	Items []Cargo
	Total int
}

// CreateParams for creating cargo with route points and payment.
type CreateParams struct {
	Weight        float64
	Volume        float64
	ReadyEnabled  bool
	ReadyAt       *string
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
	RoutePoints   []RoutePointInput
	Payment       *PaymentInput
	// Кто создал (admin/dispatcher) — заполняется из JWT при создании
	CreatedByType *string
	CreatedByID   *uuid.UUID
	CompanyID     *uuid.UUID
}

type RoutePointInput struct {
	Type         string
	CityCode     string
	RegionCode   string
	Address      string
	Orientir     string
	Lat          float64
	Lng          float64
	Comment      *string
	PointOrder   int
	IsMainLoad   bool
	IsMainUnload bool
}

type PaymentInput struct {
	IsNegotiable       bool
	PriceRequest       bool
	TotalAmount        *float64
	TotalCurrency      *string
	WithPrepayment     bool
	WithoutPrepayment  bool
	PrepaymentAmount   *float64
	PrepaymentCurrency *string
	PrepaymentType     *string
	RemainingAmount    *float64
	RemainingCurrency  *string
	RemainingType      *string
}

// Create creates cargo, route_points and payment in a transaction.
func (r *Repo) Create(ctx context.Context, p CreateParams) (uuid.UUID, error) {
	tx, err := r.pg.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	docJSON, _ := DocumentsToJSON(p.Documents)
	var id uuid.UUID
	q := `
INSERT INTO cargo (weight, volume, ready_enabled, ready_at, load_comment, truck_type,
  temp_min, temp_max, adr_enabled, adr_class, loading_types, requirements, shipment_type, belts_count,
  documents, contact_name, contact_phone, status, created_at, updated_at, deleted_at, created_by_type, created_by_id, company_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, COALESCE(NULLIF($18,''), 'created'), now(), now(), NULL, $19, $20, $21)
RETURNING id`
	err = tx.QueryRow(ctx, q,
		p.Weight, p.Volume, p.ReadyEnabled, p.ReadyAt, p.LoadComment, p.TruckType,
		p.TempMin, p.TempMax, p.ADREnabled, p.ADRClass, p.LoadingTypes, p.Requirements, p.ShipmentType, p.BeltsCount,
		docJSON, p.ContactName, p.ContactPhone, p.Status,
		p.CreatedByType, p.CreatedByID, p.CompanyID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	for _, rp := range p.RoutePoints {
		_, err = tx.Exec(ctx, `
INSERT INTO route_points (cargo_id, type, city_code, region_code, address, orientir, lat, lng, comment, point_order, is_main_load, is_main_unload)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			id, rp.Type, emptyToNil(rp.CityCode), emptyToNil(rp.RegionCode), rp.Address, emptyToNil(rp.Orientir), rp.Lat, rp.Lng, rp.Comment, rp.PointOrder, rp.IsMainLoad, rp.IsMainUnload)
		if err != nil {
			return uuid.Nil, err
		}
	}

	if p.Payment != nil {
		_, err = tx.Exec(ctx, `
INSERT INTO payments (cargo_id, is_negotiable, price_request, total_amount, total_currency, with_prepayment, without_prepayment,
  prepayment_amount, prepayment_currency, prepayment_type, remaining_amount, remaining_currency, remaining_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			id, p.Payment.IsNegotiable, p.Payment.PriceRequest, p.Payment.TotalAmount, p.Payment.TotalCurrency,
			p.Payment.WithPrepayment, p.Payment.WithoutPrepayment, p.Payment.PrepaymentAmount, p.Payment.PrepaymentCurrency,
			p.Payment.PrepaymentType, p.Payment.RemainingAmount, p.Payment.RemainingCurrency, p.Payment.RemainingType)
		if err != nil {
			return uuid.Nil, err
		}
	}

	return id, tx.Commit(ctx)
}

// GetByID returns cargo by id (excluding soft-deleted if needAll=false).
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*Cargo, error) {
	q := `SELECT id, weight, volume, ready_enabled, ready_at, load_comment, truck_type,
  temp_min, temp_max, adr_enabled, adr_class, loading_types, requirements, shipment_type, belts_count,
  documents, contact_name, contact_phone, status, created_at, updated_at, deleted_at, moderation_rejection_reason, created_by_type, created_by_id, company_id
FROM cargo WHERE id = $1`
	if !includeDeleted {
		q += ` AND deleted_at IS NULL`
	}
	return scanCargo(r.pg.QueryRow(ctx, q, id))
}

// GetRoutePoints returns route points for a cargo.
func (r *Repo) GetRoutePoints(ctx context.Context, cargoID uuid.UUID) ([]RoutePoint, error) {
	rows, err := r.pg.Query(ctx, `
SELECT id, cargo_id, type, COALESCE(city_code,''), COALESCE(region_code,''), address, COALESCE(orientir,''), lat, lng, comment, point_order, is_main_load, is_main_unload
FROM route_points WHERE cargo_id = $1 ORDER BY point_order`,
		cargoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []RoutePoint
	for rows.Next() {
		var rp RoutePoint
		err := rows.Scan(&rp.ID, &rp.CargoID, &rp.Type, &rp.CityCode, &rp.RegionCode, &rp.Address, &rp.Orientir, &rp.Lat, &rp.Lng, &rp.Comment, &rp.PointOrder, &rp.IsMainLoad, &rp.IsMainUnload)
		if err != nil {
			return nil, err
		}
		list = append(list, rp)
	}
	return list, rows.Err()
}

// GetPayment returns payment for a cargo (if any).
func (r *Repo) GetPayment(ctx context.Context, cargoID uuid.UUID) (*Payment, error) {
	var pay Payment
	err := r.pg.QueryRow(ctx, `
SELECT id, cargo_id, is_negotiable, price_request, total_amount, total_currency, with_prepayment, without_prepayment,
  prepayment_amount, prepayment_currency, prepayment_type, remaining_amount, remaining_currency, remaining_type
FROM payments WHERE cargo_id = $1`, cargoID).Scan(
		&pay.ID, &pay.CargoID, &pay.IsNegotiable, &pay.PriceRequest, &pay.TotalAmount, &pay.TotalCurrency,
		&pay.WithPrepayment, &pay.WithoutPrepayment, &pay.PrepaymentAmount, &pay.PrepaymentCurrency,
		&pay.PrepaymentType, &pay.RemainingAmount, &pay.RemainingCurrency, &pay.RemainingType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pay, nil
}

func scanCargo(row pgx.Row) (*Cargo, error) {
	var c Cargo
	var docBytes []byte
	var loadingTypes, requirements []string
	err := row.Scan(
		&c.ID, &c.Weight, &c.Volume, &c.ReadyEnabled, &c.ReadyAt, &c.LoadComment, &c.TruckType,
		&c.TempMin, &c.TempMax, &c.ADREnabled, &c.ADRClass, &loadingTypes, &requirements, &c.ShipmentType, &c.BeltsCount,
		&docBytes, &c.ContactName, &c.ContactPhone, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		&c.ModerationRejectionReason, &c.CreatedByType, &c.CreatedByID, &c.CompanyID,
	)
	if err != nil {
		return nil, err
	}
	c.LoadingTypes = loadingTypes
	c.Requirements = requirements
	if len(docBytes) > 0 {
		c.Documents, _ = DocumentsFromJSON(docBytes)
	}
	return &c, nil
}

// List returns paginated cargo list with filters.
func (r *Repo) List(ctx context.Context, f ListFilter) (ListResult, error) {
	var args []any
	var conds []string
	argNum := 1
	conds = append(conds, "deleted_at IS NULL")

	if len(f.Status) > 0 {
		conds = append(conds, "status = ANY($"+strconv.Itoa(argNum)+")")
		args = append(args, f.Status)
		argNum++
	}
	if f.WeightMin != nil {
		conds = append(conds, "weight >= $"+strconv.Itoa(argNum))
		args = append(args, *f.WeightMin)
		argNum++
	}
	if f.WeightMax != nil {
		conds = append(conds, "weight <= $"+strconv.Itoa(argNum))
		args = append(args, *f.WeightMax)
		argNum++
	}
	if f.TruckType != "" {
		conds = append(conds, "truck_type = $"+strconv.Itoa(argNum))
		args = append(args, f.TruckType)
		argNum++
	}
	if f.CreatedFrom != "" {
		conds = append(conds, "created_at::date >= $"+strconv.Itoa(argNum))
		args = append(args, f.CreatedFrom)
		argNum++
	}
	if f.CreatedTo != "" {
		conds = append(conds, "created_at::date <= $"+strconv.Itoa(argNum))
		args = append(args, f.CreatedTo)
		argNum++
	}
	if f.WithOffers != nil && *f.WithOffers {
		conds = append(conds, "EXISTS (SELECT 1 FROM offers o WHERE o.cargo_id = cargo.id)")
	}

	where := strings.Join(conds, " AND ")

	// total
	var total int
	err := r.pg.QueryRow(ctx, "SELECT COUNT(*) FROM cargo WHERE "+where, args...).Scan(&total)
	if err != nil {
		return ListResult{}, err
	}

	order := "created_at DESC"
	if f.Sort != "" {
		parts := strings.SplitN(f.Sort, ":", 2)
		if len(parts) == 2 {
			col := strings.TrimSpace(parts[0])
			dir := strings.ToUpper(strings.TrimSpace(parts[1]))
			if col == "created_at" || col == "weight" || col == "status" {
				if dir == "ASC" || dir == "DESC" {
					order = col + " " + dir
				}
			}
		}
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (f.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	q := `SELECT id, weight, volume, ready_enabled, ready_at, load_comment, truck_type,
  temp_min, temp_max, adr_enabled, adr_class, loading_types, requirements, shipment_type, belts_count,
  documents, contact_name, contact_phone, status, created_at, updated_at, deleted_at, moderation_rejection_reason, created_by_type, created_by_id, company_id
FROM cargo WHERE ` + where + ` ORDER BY ` + order + ` LIMIT $` + strconv.Itoa(argNum) + ` OFFSET $` + strconv.Itoa(argNum+1)

	rows, err := r.pg.Query(ctx, q, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	var items []Cargo
	for rows.Next() {
		c, err := scanCargo(rows)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, *c)
	}
	return ListResult{Items: items, Total: total}, rows.Err()
}

// nextArgNum returns placeholder number as string (1, 2, ...).
func nextArgNum(n *int) string {
	x := *n
	*n++
	return strconv.Itoa(x)
}

// UpdateParams for PUT /api/cargo/:id (partial; only non-nil fields updated where applicable).
type UpdateParams struct {
	Weight        *float64
	Volume        *float64
	ReadyEnabled  *bool
	ReadyAt       *string
	LoadComment   *string
	TruckType     *string
	TempMin       *float64
	TempMax       *float64
	ADREnabled    *bool
	ADRClass      *string
	LoadingTypes  []string
	Requirements  []string
	ShipmentType  *string
	BeltsCount    *int
	Documents     *Documents
	ContactName   *string
	ContactPhone  *string
	RoutePoints   []RoutePointInput
	Payment       *PaymentInput
}

// Update updates cargo and optionally replaces route_points and payment. Returns error if cargo not found or deleted.
func (r *Repo) Update(ctx context.Context, id uuid.UUID, p UpdateParams) error {
	tx, err := r.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	existing, err := r.GetByID(ctx, id, false)
	if err != nil || existing == nil {
		return err
	}
	if existing.Status == StatusAssigned || existing.Status == StatusInTransit || existing.Status == StatusDelivered {
		// After assigned cannot change price and route - we block full update of route and payment
		// but allow contact/comment edits if needed; for simplicity we block any update of route_points
		if len(p.RoutePoints) > 0 || p.Payment != nil {
			return ErrCannotEditAfterAssigned
		}
	}

	// Build dynamic update for cargo
	setCols := []string{"updated_at = now()"}
	args := []any{}
	argN := 1
	add := func(col string, v any) {
		setCols = append(setCols, col+" = $"+nextArgNum(&argN))
		args = append(args, v)
	}
	if p.Weight != nil {
		add("weight", *p.Weight)
	}
	if p.Volume != nil {
		add("volume", *p.Volume)
	}
	if p.ReadyEnabled != nil {
		add("ready_enabled", *p.ReadyEnabled)
	}
	if p.ReadyAt != nil {
		add("ready_at", p.ReadyAt)
	}
	if p.LoadComment != nil {
		add("load_comment", *p.LoadComment)
	}
	if p.TruckType != nil {
		add("truck_type", *p.TruckType)
	}
	if p.TempMin != nil {
		add("temp_min", *p.TempMin)
	}
	if p.TempMax != nil {
		add("temp_max", *p.TempMax)
	}
	if p.ADREnabled != nil {
		add("adr_enabled", *p.ADREnabled)
	}
	if p.ADRClass != nil {
		add("adr_class", *p.ADRClass)
	}
	if p.LoadingTypes != nil {
		add("loading_types", p.LoadingTypes)
	}
	if p.Requirements != nil {
		add("requirements", p.Requirements)
	}
	if p.ShipmentType != nil {
		add("shipment_type", *p.ShipmentType)
	}
	if p.BeltsCount != nil {
		add("belts_count", *p.BeltsCount)
	}
	if p.Documents != nil {
		docJSON, _ := DocumentsToJSON(p.Documents)
		add("documents", docJSON)
	}
	if p.ContactName != nil {
		add("contact_name", *p.ContactName)
	}
	if p.ContactPhone != nil {
		add("contact_phone", *p.ContactPhone)
	}

	if len(setCols) > 1 {
		args = append(args, id)
		_, err = tx.Exec(ctx, "UPDATE cargo SET "+strings.Join(setCols, ", ")+" WHERE id = $"+nextArgNum(&argN)+" AND deleted_at IS NULL", args...)
		if err != nil {
			return err
		}
	}

	if len(p.RoutePoints) > 0 && existing.Status != StatusAssigned && existing.Status != StatusInTransit && existing.Status != StatusDelivered {
		_, _ = tx.Exec(ctx, "DELETE FROM route_points WHERE cargo_id = $1", id)
		for _, rp := range p.RoutePoints {
			_, err = tx.Exec(ctx, `
INSERT INTO route_points (cargo_id, type, city_code, region_code, address, orientir, lat, lng, comment, point_order, is_main_load, is_main_unload)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				id, rp.Type, emptyToNil(rp.CityCode), emptyToNil(rp.RegionCode), rp.Address, emptyToNil(rp.Orientir), rp.Lat, rp.Lng, rp.Comment, rp.PointOrder, rp.IsMainLoad, rp.IsMainUnload)
			if err != nil {
				return err
			}
		}
	}

	if p.Payment != nil && existing.Status != StatusAssigned && existing.Status != StatusInTransit && existing.Status != StatusDelivered {
		_, err = tx.Exec(ctx, `
UPDATE payments SET is_negotiable=$2, price_request=$3, total_amount=$4, total_currency=$5, with_prepayment=$6, without_prepayment=$7,
  prepayment_amount=$8, prepayment_currency=$9, prepayment_type=$10, remaining_amount=$11, remaining_currency=$12, remaining_type=$13
WHERE cargo_id = $1`,
			id, p.Payment.IsNegotiable, p.Payment.PriceRequest, p.Payment.TotalAmount, p.Payment.TotalCurrency,
			p.Payment.WithPrepayment, p.Payment.WithoutPrepayment, p.Payment.PrepaymentAmount, p.Payment.PrepaymentCurrency,
			p.Payment.PrepaymentType, p.Payment.RemainingAmount, p.Payment.RemainingCurrency, p.Payment.RemainingType)
		if err != nil {
			return err
		}
		// If no row updated, insert
		var n int
		_ = tx.QueryRow(ctx, "SELECT 1 FROM payments WHERE cargo_id = $1", id).Scan(&n)
		if n == 0 {
			_, err = tx.Exec(ctx, `
INSERT INTO payments (cargo_id, is_negotiable, price_request, total_amount, total_currency, with_prepayment, without_prepayment,
  prepayment_amount, prepayment_currency, prepayment_type, remaining_amount, remaining_currency, remaining_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
				id, p.Payment.IsNegotiable, p.Payment.PriceRequest, p.Payment.TotalAmount, p.Payment.TotalCurrency,
				p.Payment.WithPrepayment, p.Payment.WithoutPrepayment, p.Payment.PrepaymentAmount, p.Payment.PrepaymentCurrency,
				p.Payment.PrepaymentType, p.Payment.RemainingAmount, p.Payment.RemainingCurrency, p.Payment.RemainingType)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

var ErrCannotEditAfterAssigned = errors.New("cargo: cannot edit route or payment after assigned")

// CountByDispatcher возвращает число грузов, созданных диспетчером (created_by_type='dispatcher', без удалённых).
func (r *Repo) CountByDispatcher(ctx context.Context, dispatcherID uuid.UUID) (int, error) {
	var n int
	err := r.pg.QueryRow(ctx,
		"SELECT count(*) FROM cargo WHERE created_by_type = 'dispatcher' AND created_by_id = $1 AND deleted_at IS NULL",
		dispatcherID).Scan(&n)
	return n, err
}

// Delete soft-deletes cargo (sets deleted_at).
func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pg.Exec(ctx, "UPDATE cargo SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL", id)
	return err
}

// SetStatus updates cargo status with allowed transitions. Returns error if transition invalid.
func (r *Repo) SetStatus(ctx context.Context, id uuid.UUID, newStatus string) error {
	allowed := map[string][]string{
		StatusCreated:           {StatusSearching, StatusCancelled},
		StatusPendingModeration: {StatusSearching, StatusRejected},
		StatusSearching:         {StatusAssigned, StatusCancelled},
		StatusRejected:          nil,
		StatusAssigned:          {StatusInProgress, StatusCancelled},
		StatusInProgress:        {StatusCompleted},
		StatusInTransit:         {StatusDelivered},
		StatusDelivered:         nil,
		StatusCompleted:         nil,
		StatusCancelled:         nil,
	}
	cur, err := r.GetByID(ctx, id, false)
	if err != nil || cur == nil {
		return err
	}
	next, ok := allowed[cur.Status]
	if !ok {
		return errors.New("cargo: invalid current status")
	}
	for _, s := range next {
		if s == newStatus {
			_, err = r.pg.Exec(ctx, "UPDATE cargo SET status = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL", newStatus, id)
			return err
		}
	}
	return errors.New("cargo: status transition not allowed")
}

// GetOfferByID returns one offer by id (nil if not found).
func (r *Repo) GetOfferByID(ctx context.Context, offerID uuid.UUID) (*Offer, error) {
	var o Offer
	var rejReason string
	err := r.pg.QueryRow(ctx, `
SELECT id, cargo_id, carrier_id, price, currency, comment, status, COALESCE(rejection_reason, ''), created_at
FROM offers WHERE id = $1`, offerID).Scan(&o.ID, &o.CargoID, &o.CarrierID, &o.Price, &o.Currency, &o.Comment, &o.Status, &rejReason, &o.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if rejReason != "" {
		o.RejectionReason = &rejReason
	}
	return &o, nil
}

// GetOffers returns all offers for a cargo.
func (r *Repo) GetOffers(ctx context.Context, cargoID uuid.UUID) ([]Offer, error) {
	rows, err := r.pg.Query(ctx, `
SELECT id, cargo_id, carrier_id, price, currency, comment, status, COALESCE(rejection_reason, ''), created_at
FROM offers WHERE cargo_id = $1 ORDER BY created_at DESC`, cargoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Offer
	for rows.Next() {
		var o Offer
		var rejReason string
		err := rows.Scan(&o.ID, &o.CargoID, &o.CarrierID, &o.Price, &o.Currency, &o.Comment, &o.Status, &rejReason, &o.CreatedAt)
		if err != nil {
			return nil, err
		}
		if rejReason != "" {
			o.RejectionReason = &rejReason
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// CreateOffer inserts an offer for a cargo.
func (r *Repo) CreateOffer(ctx context.Context, cargoID, carrierID uuid.UUID, price float64, currency, comment string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pg.QueryRow(ctx, `
INSERT INTO offers (cargo_id, carrier_id, price, currency, comment, status, created_at)
VALUES ($1, $2, $3, $4, $5, 'pending', now()) RETURNING id`,
		cargoID, carrierID, price, currency, nullStr(comment)).Scan(&id)
	return id, err
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// AcceptOffer sets offer status to accepted and cargo status to assigned. Returns cargoID and carrierID (driver).
func (r *Repo) AcceptOffer(ctx context.Context, offerID uuid.UUID) (cargoID, carrierID uuid.UUID, err error) {
	tx, err := r.pg.Begin(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, "SELECT cargo_id, carrier_id FROM offers WHERE id = $1 AND status = 'pending'", offerID).Scan(&cargoID, &carrierID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, uuid.Nil, errors.New("cargo: offer not found or not pending")
		}
		return uuid.Nil, uuid.Nil, err
	}
	_, err = tx.Exec(ctx, "UPDATE offers SET status = 'accepted' WHERE id = $1", offerID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	_, err = tx.Exec(ctx, "UPDATE cargo SET status = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL", StatusAssigned, cargoID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	_, _ = tx.Exec(ctx, "UPDATE offers SET status = 'rejected' WHERE cargo_id = $1 AND id != $2 AND status = 'pending'", cargoID, offerID)
	return cargoID, carrierID, tx.Commit(ctx)
}

// RejectOffer sets offer status to rejected with optional reason (dispatcher).
func (r *Repo) RejectOffer(ctx context.Context, offerID uuid.UUID, reason string) error {
	res, err := r.pg.Exec(ctx,
		"UPDATE offers SET status = 'rejected', rejection_reason = NULLIF(TRIM($2), '') WHERE id = $1 AND status = 'pending'",
		offerID, reason)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("cargo: offer not found or not pending")
	}
	return nil
}

// ModerationAccept sets cargo status to searching (admin approved).
func (r *Repo) ModerationAccept(ctx context.Context, cargoID uuid.UUID) error {
	res, err := r.pg.Exec(ctx,
		"UPDATE cargo SET status = $1, updated_at = now(), moderation_rejection_reason = NULL WHERE id = $2 AND deleted_at IS NULL AND status = $3",
		StatusSearching, cargoID, StatusPendingModeration)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("cargo: not found or not pending_moderation")
	}
	return nil
}

// ModerationReject sets cargo status to rejected with mandatory reason (admin).
func (r *Repo) ModerationReject(ctx context.Context, cargoID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("cargo: moderation rejection reason is required")
	}
	res, err := r.pg.Exec(ctx,
		"UPDATE cargo SET status = $1, moderation_rejection_reason = $2, updated_at = now() WHERE id = $3 AND deleted_at IS NULL AND status = $4",
		StatusRejected, strings.TrimSpace(reason), cargoID, StatusPendingModeration)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("cargo: not found or not pending_moderation")
	}
	return nil
}

// ListPendingModeration returns cargo list with status pending_moderation (for admin).
func (r *Repo) ListPendingModeration(ctx context.Context, limit, offset int) ([]Cargo, int, error) {
	var total int
	_ = r.pg.QueryRow(ctx, "SELECT count(*) FROM cargo WHERE deleted_at IS NULL AND status = $1", StatusPendingModeration).Scan(&total)
	q := `SELECT id, weight, volume, ready_enabled, ready_at, load_comment, truck_type,
  temp_min, temp_max, adr_enabled, adr_class, loading_types, requirements, shipment_type, belts_count,
  documents, contact_name, contact_phone, status, created_at, updated_at, deleted_at, moderation_rejection_reason, created_by_type, created_by_id, company_id
FROM cargo WHERE deleted_at IS NULL AND status = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`
	rows, err := r.pg.Query(ctx, q, StatusPendingModeration, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []Cargo
	for rows.Next() {
		c, err := scanCargo(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *c)
	}
	return list, total, rows.Err()
}

// SetCargoStatusInProgress sets cargo status to in_progress (when trip execution starts).
func (r *Repo) SetCargoStatusInProgress(ctx context.Context, cargoID uuid.UUID) error {
	res, err := r.pg.Exec(ctx,
		"UPDATE cargo SET status = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL AND status = $3",
		StatusInProgress, cargoID, StatusAssigned)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return nil // already in progress or other status, idempotent
	}
	return nil
}

// SetCargoStatusCompleted sets cargo status to completed (when trip is completed).
func (r *Repo) SetCargoStatusCompleted(ctx context.Context, cargoID uuid.UUID) error {
	_, err := r.pg.Exec(ctx,
		"UPDATE cargo SET status = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL AND (status = $3 OR status = $4)",
		StatusCompleted, cargoID, StatusInProgress, StatusInTransit)
	return err
}

// emptyToNil returns nil for empty string (for NULL in DB), else the string.
func emptyToNil(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
