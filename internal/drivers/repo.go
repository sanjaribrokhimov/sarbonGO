package drivers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sarbonNew/internal/util"
)

var ErrNotFound = errors.New("driver not found")

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

const driverSelectCols = `
  d.id, d.phone, d.created_at, d.updated_at, d.last_online_at, d.latitude, d.longitude, d.push_token,
  d.registration_step, d.registration_status, d.name, d.driver_type, d.rating, d.work_status,
  d.freelancer_id, d.company_id, d.account_status,
  d.driver_passport_series, d.driver_passport_number, d.driver_pinfl, d.driver_scan_status,
  p.power_plate_type, p.power_plate_number, p.power_tech_series, p.power_tech_number, p.power_owner_id, p.power_owner_name, p.power_scan_status,
  t.trailer_plate_type, t.trailer_plate_number, t.trailer_tech_series, t.trailer_tech_number, t.trailer_owner_id, t.trailer_owner_name, t.trailer_scan_status,
  d.driver_owner, d.kyc_status,
  (d.photo_data IS NOT NULL) AS has_photo`

const driverJoinTables = `
FROM drivers d
LEFT JOIN driver_powers p ON p.driver_id = d.id
LEFT JOIN driver_trailers t ON t.driver_id = d.id`

func (r *Repo) FindByPhone(ctx context.Context, phone string) (*Driver, error) {
	const q = `SELECT ` + driverSelectCols + driverJoinTables + ` WHERE d.phone = $1 LIMIT 1`
	d, err := scanDriver(r.pg.QueryRow(ctx, q, phone))
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *Repo) FindByID(ctx context.Context, id uuid.UUID) (*Driver, error) {
	const q = `SELECT ` + driverSelectCols + driverJoinTables + ` WHERE d.id = $1 LIMIT 1`
	d, err := scanDriver(r.pg.QueryRow(ctx, q, id))
	if err != nil {
		return nil, err
	}
	return d, nil
}

func scanDriver(row pgx.Row) (*Driver, error) {
	var d Driver
	err := row.Scan(
		&d.ID, &d.Phone, &d.CreatedAt, &d.UpdatedAt, &d.LastOnlineAt, &d.Latitude, &d.Longitude, &d.PushToken,
		&d.RegistrationStep, &d.RegistrationStatus, &d.Name, &d.DriverType, &d.Rating, &d.WorkStatus,
		&d.FreelancerID, &d.CompanyID, &d.AccountStatus,
		&d.DriverPassportSeries, &d.DriverPassportNumber, &d.DriverPINFL, &d.DriverScanStatus,
		&d.PowerPlateType, &d.PowerPlateNumber, &d.PowerTechSeries, &d.PowerTechNumber, &d.PowerOwnerID, &d.PowerOwnerName, &d.PowerScanStatus,
		&d.TrailerPlateType, &d.TrailerPlateNumber, &d.TrailerTechSeries, &d.TrailerTechNumber, &d.TrailerOwnerID, &d.TrailerOwnerName, &d.TrailerScanStatus,
		&d.DriverOwner, &d.KYCStatus,
		&d.HasPhoto,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	d.CreatedAt = util.InTashkent(d.CreatedAt)
	d.UpdatedAt = util.InTashkent(d.UpdatedAt)
	if d.LastOnlineAt != nil {
		v := util.InTashkent(*d.LastOnlineAt)
		d.LastOnlineAt = &v
	}

	return &d, nil
}

func (r *Repo) CreateStart(ctx context.Context, phone string, ownerName string) (uuid.UUID, error) {
	const q = `
INSERT INTO drivers (phone, name, registration_status, registration_step, created_at, updated_at, last_online_at)
VALUES ($1, $2, $3, $4, now(), now(), now())
RETURNING id`
	var id uuid.UUID
	err := r.pg.QueryRow(ctx, q, phone, ownerName, "start", "name-oferta").Scan(&id)
	return id, err
}

func (r *Repo) TouchOnline(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE drivers SET last_online_at = now(), updated_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id)
	return err
}

type UpdateDriverEditable struct {
	Name                 *string
	WorkStatus           *string // available|loaded|busy
	DriverPassportSeries *string
	DriverPassportNumber *string
	DriverPINFL          *string
}

func (r *Repo) UpdateDriverEditable(ctx context.Context, id uuid.UUID, u UpdateDriverEditable) error {
	const q = `
UPDATE drivers
SET name = COALESCE($2, name),
    work_status = COALESCE($3, work_status),
    driver_passport_series = COALESCE($4, driver_passport_series),
    driver_passport_number = COALESCE($5, driver_passport_number),
    driver_pinfl = COALESCE($6, driver_pinfl),
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, u.Name, u.WorkStatus, u.DriverPassportSeries, u.DriverPassportNumber, u.DriverPINFL)
	return err
}

// SetCompanyID sets driver's company (e.g. after accepting company invitation).
func (r *Repo) SetCompanyID(ctx context.Context, driverID, companyID uuid.UUID) error {
	const q = `UPDATE drivers SET company_id = $2, updated_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, driverID, companyID)
	return err
}

// SetFreelancerID sets driver's freelancer (dispatcher) — e.g. after accepting freelance dispatcher invitation.
func (r *Repo) SetFreelancerID(ctx context.Context, driverID, freelancerID uuid.UUID) error {
	const q = `UPDATE drivers SET freelancer_id = $2, updated_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, driverID, freelancerID)
	return err
}

// UnlinkFromFreelancer removes driver from dispatcher (sets freelancer_id = NULL). Only if driver is currently linked to this freelancer.
func (r *Repo) UnlinkFromFreelancer(ctx context.Context, driverID, freelancerID uuid.UUID) (bool, error) {
	const q = `UPDATE drivers SET freelancer_id = NULL, updated_at = now() WHERE id = $1 AND freelancer_id = $2`
	tag, err := r.pg.Exec(ctx, q, driverID, freelancerID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SearchByPhone returns drivers whose phone matches the search (exact match first, then containing). For dispatcher to find driver and invite by id.
func (r *Repo) SearchByPhone(ctx context.Context, phoneSearch string, limit int) ([]*Driver, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	term := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(phoneSearch, " ", ""), "-", ""))
	if term == "" {
		return []*Driver{}, nil
	}
	pattern := "%" + term + "%"
	const q = `SELECT ` + driverSelectCols + driverJoinTables + `
WHERE replace(replace(trim(d.phone), ' ', ''), '-', '') LIKE $1
ORDER BY (replace(replace(trim(d.phone), ' ', ''), '-', '') = $2) DESC, d.created_at DESC
LIMIT $3`
	rows, err := r.pg.Query(ctx, q, pattern, term, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Driver
	for rows.Next() {
		d, err := scanDriver(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// ListByFreelancerID returns drivers linked to this freelance dispatcher (freelancer_id = dispatcherID).
func (r *Repo) ListByFreelancerID(ctx context.Context, freelancerID uuid.UUID, limit int) ([]*Driver, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `SELECT ` + driverSelectCols + driverJoinTables + ` WHERE d.freelancer_id = $1 ORDER BY d.updated_at DESC LIMIT $2`
	rows, err := r.pg.Query(ctx, q, freelancerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Driver
	for rows.Next() {
		d, err := scanDriver(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *Repo) UpdateHeartbeat(ctx context.Context, id uuid.UUID, lat, lon float64, lastOnlineAt time.Time) error {
	const q = `
UPDATE drivers
SET latitude = $2,
    longitude = $3,
    last_online_at = $4,
    updated_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, lat, lon, lastOnlineAt)
	return err
}

var ErrPhoneAlreadyRegistered = errors.New("phone already registered")

func (r *Repo) UpdatePhone(ctx context.Context, id uuid.UUID, newPhone string) error {
	const q = `UPDATE drivers SET phone = $2, updated_at = now(), last_online_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, newPhone)
	if err != nil {
		// detect unique violation
		// pgx returns *pgconn.PgError for server-side errors
		type pgErr interface{ SQLState() string }
		if e, ok := err.(pgErr); ok && e.SQLState() == "23505" {
			return ErrPhoneAlreadyRegistered
		}
		return err
	}
	return nil
}

type UpdatePowerProfile struct {
	PowerPlateType   *string
	PowerPlateNumber *string
	PowerTechSeries  *string
	PowerTechNumber  *string
	PowerOwnerID     *string
	PowerOwnerName   *string
	PowerScanStatus  *bool
}

func (r *Repo) UpdatePowerProfile(ctx context.Context, id uuid.UUID, u UpdatePowerProfile) error {
	const q = `
INSERT INTO driver_powers (driver_id, power_plate_type, power_plate_number, power_tech_series, power_tech_number, power_owner_id, power_owner_name, power_scan_status, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (driver_id) DO UPDATE SET
  power_plate_type = COALESCE(EXCLUDED.power_plate_type, driver_powers.power_plate_type),
  power_plate_number = COALESCE(EXCLUDED.power_plate_number, driver_powers.power_plate_number),
  power_tech_series = COALESCE(EXCLUDED.power_tech_series, driver_powers.power_tech_series),
  power_tech_number = COALESCE(EXCLUDED.power_tech_number, driver_powers.power_tech_number),
  power_owner_id = COALESCE(EXCLUDED.power_owner_id, driver_powers.power_owner_id),
  power_owner_name = COALESCE(EXCLUDED.power_owner_name, driver_powers.power_owner_name),
  power_scan_status = COALESCE(EXCLUDED.power_scan_status, driver_powers.power_scan_status),
  updated_at = now()`
	_, err := r.pg.Exec(ctx, q, id, u.PowerPlateType, u.PowerPlateNumber, u.PowerTechSeries, u.PowerTechNumber, u.PowerOwnerID, u.PowerOwnerName, u.PowerScanStatus)
	if err != nil {
		return err
	}
	_, err = r.pg.Exec(ctx, `UPDATE drivers SET updated_at = now(), last_online_at = now() WHERE id = $1`, id)
	return err
}

type UpdateTrailerProfile struct {
	TrailerPlateType   *string
	TrailerPlateNumber *string
	TrailerTechSeries  *string
	TrailerTechNumber  *string
	TrailerOwnerID     *string
	TrailerOwnerName   *string
	TrailerScanStatus  *bool
}

func (r *Repo) UpdateTrailerProfile(ctx context.Context, id uuid.UUID, u UpdateTrailerProfile) error {
	const q = `
INSERT INTO driver_trailers (driver_id, trailer_plate_type, trailer_plate_number, trailer_tech_series, trailer_tech_number, trailer_owner_id, trailer_owner_name, trailer_scan_status, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (driver_id) DO UPDATE SET
  trailer_plate_type = COALESCE(EXCLUDED.trailer_plate_type, driver_trailers.trailer_plate_type),
  trailer_plate_number = COALESCE(EXCLUDED.trailer_plate_number, driver_trailers.trailer_plate_number),
  trailer_tech_series = COALESCE(EXCLUDED.trailer_tech_series, driver_trailers.trailer_tech_series),
  trailer_tech_number = COALESCE(EXCLUDED.trailer_tech_number, driver_trailers.trailer_tech_number),
  trailer_owner_id = COALESCE(EXCLUDED.trailer_owner_id, driver_trailers.trailer_owner_id),
  trailer_owner_name = COALESCE(EXCLUDED.trailer_owner_name, driver_trailers.trailer_owner_name),
  trailer_scan_status = COALESCE(EXCLUDED.trailer_scan_status, driver_trailers.trailer_scan_status),
  updated_at = now()`
	_, err := r.pg.Exec(ctx, q, id, u.TrailerPlateType, u.TrailerPlateNumber, u.TrailerTechSeries, u.TrailerTechNumber, u.TrailerOwnerID, u.TrailerOwnerName, u.TrailerScanStatus)
	if err != nil {
		return err
	}
	_, err = r.pg.Exec(ctx, `UPDATE drivers SET updated_at = now(), last_online_at = now() WHERE id = $1`, id)
	return err
}

var ErrDeleteNotFound = errors.New("driver to delete not found")

func (r *Repo) DeleteAndArchive(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `INSERT INTO deleted_drivers SELECT * FROM drivers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDeleteNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM drivers WHERE id = $1`, id); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Repo) UpdateGeo(ctx context.Context, id uuid.UUID, lat, lon float64, nextStep string, pushToken *string) error {
	const q = `
UPDATE drivers
SET latitude = $2,
    longitude = $3,
    registration_step = $4,
    push_token = COALESCE($5, push_token),
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, lat, lon, nextStep, pushToken)
	return err
}

func (r *Repo) UpdatePushToken(ctx context.Context, id uuid.UUID, pushToken string) error {
	const q = `
UPDATE drivers
SET push_token = $2,
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, pushToken)
	return err
}

func (r *Repo) UpdateTransportType(ctx context.Context, id uuid.UUID, driverType string, freelancerID *uuid.UUID, companyID *uuid.UUID, powerPlateType string, trailerPlateType string, nextStep string, nextStatus string) error {
	_, err := r.pg.Exec(ctx, `
UPDATE drivers
SET driver_type = $2, freelancer_id = $3, company_id = $4, registration_step = $5, registration_status = $6, updated_at = now(), last_online_at = now()
WHERE id = $1`, id, driverType, freelancerID, companyID, nextStep, nextStatus)
	if err != nil {
		return err
	}
	_, err = r.pg.Exec(ctx, `
INSERT INTO driver_powers (driver_id, power_plate_type, updated_at) VALUES ($1, $2, now())
ON CONFLICT (driver_id) DO UPDATE SET power_plate_type = EXCLUDED.power_plate_type, updated_at = now()`, id, powerPlateType)
	if err != nil {
		return err
	}
	_, err = r.pg.Exec(ctx, `
INSERT INTO driver_trailers (driver_id, trailer_plate_type, updated_at) VALUES ($1, $2, now())
ON CONFLICT (driver_id) DO UPDATE SET trailer_plate_type = EXCLUDED.trailer_plate_type, updated_at = now()`, id, trailerPlateType)
	return err
}

type KYCUpdate struct {
	DriverPassportSeries string
	DriverPassportNumber string
	DriverPINFL          string
	DriverScanStatus     *bool

	PowerPlateNumber   string
	PowerTechSeries    string
	PowerTechNumber    string
	PowerOwnerID       string
	PowerOwnerName     string
	PowerScanStatus    *bool

	TrailerPlateNumber string
	TrailerTechSeries  string
	TrailerTechNumber  string
	TrailerOwnerID     string
	TrailerOwnerName   string
	TrailerScanStatus  *bool

	DriverOwner *bool
	KYCStatus     string

	RegistrationStatus string
	RegistrationStep  string // после KYC — "completed"
}

func (r *Repo) UpdateKYC(ctx context.Context, id uuid.UUID, u KYCUpdate) error {
	_, err := r.pg.Exec(ctx, `
UPDATE drivers
SET driver_passport_series = $2, driver_passport_number = $3, driver_pinfl = $4, driver_scan_status = $5,
    driver_owner = $6, kyc_status = $7, registration_status = $8, registration_step = $9,
    updated_at = now(), last_online_at = now()
WHERE id = $1`,
		id, u.DriverPassportSeries, u.DriverPassportNumber, u.DriverPINFL, u.DriverScanStatus,
		u.DriverOwner, u.KYCStatus, u.RegistrationStatus, u.RegistrationStep,
	)
	if err != nil {
		return err
	}
	_, err = r.pg.Exec(ctx, `
INSERT INTO driver_powers (driver_id, power_plate_number, power_tech_series, power_tech_number, power_owner_id, power_owner_name, power_scan_status, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (driver_id) DO UPDATE SET
  power_plate_number = EXCLUDED.power_plate_number,
  power_tech_series = EXCLUDED.power_tech_series,
  power_tech_number = EXCLUDED.power_tech_number,
  power_owner_id = EXCLUDED.power_owner_id,
  power_owner_name = EXCLUDED.power_owner_name,
  power_scan_status = EXCLUDED.power_scan_status,
  updated_at = now()`,
		id, u.PowerPlateNumber, u.PowerTechSeries, u.PowerTechNumber, u.PowerOwnerID, u.PowerOwnerName, u.PowerScanStatus,
	)
	if err != nil {
		return err
	}
	_, err = r.pg.Exec(ctx, `
INSERT INTO driver_trailers (driver_id, trailer_plate_number, trailer_tech_series, trailer_tech_number, trailer_owner_id, trailer_owner_name, trailer_scan_status, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (driver_id) DO UPDATE SET
  trailer_plate_number = EXCLUDED.trailer_plate_number,
  trailer_tech_series = EXCLUDED.trailer_tech_series,
  trailer_tech_number = EXCLUDED.trailer_tech_number,
  trailer_owner_id = EXCLUDED.trailer_owner_id,
  trailer_owner_name = EXCLUDED.trailer_owner_name,
  trailer_scan_status = EXCLUDED.trailer_scan_status,
  updated_at = now()`,
		id, u.TrailerPlateNumber, u.TrailerTechSeries, u.TrailerTechNumber, u.TrailerOwnerID, u.TrailerOwnerName, u.TrailerScanStatus,
	)
	return err
}

func (r *Repo) ApplyFullDefaults(ctx context.Context, id uuid.UUID) error {
	const q = `
UPDATE drivers
SET rating = COALESCE(rating, 0),
    work_status = COALESCE(work_status, 'available'),
    account_status = COALESCE(account_status, 'active'),
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id)
	return err
}

func (r *Repo) UpdateStep(ctx context.Context, id uuid.UUID, step string) error {
	const q = `UPDATE drivers SET registration_step = $2, updated_at = now(), last_online_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, step)
	return err
}

func (r *Repo) UpdateOnlineAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	const q = `UPDATE drivers SET last_online_at = $2, updated_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, t)
	return err
}

// UpdatePhoto сохраняет фото водителя в БД (бинарные данные + content-type).
func (r *Repo) UpdatePhoto(ctx context.Context, id uuid.UUID, data []byte, contentType string) error {
	const q = `UPDATE drivers SET photo_data = $2, photo_content_type = $3, updated_at = now(), last_online_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, data, contentType)
	return err
}

// GetPhoto возвращает фото водителя (данные и content-type). Если фото нет — ErrNotFound.
func (r *Repo) GetPhoto(ctx context.Context, id uuid.UUID) (data []byte, contentType string, err error) {
	const q = `SELECT photo_data, COALESCE(photo_content_type, 'image/jpeg') FROM drivers WHERE id = $1 AND photo_data IS NOT NULL`
	err = r.pg.QueryRow(ctx, q, id).Scan(&data, &contentType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	return data, contentType, nil
}

// DeletePhoto удаляет фото водителя (обнуляет photo_data).
func (r *Repo) DeletePhoto(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE drivers SET photo_data = NULL, photo_content_type = NULL, updated_at = now(), last_online_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id)
	return err
}

