package drivers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("driver not found")

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

func (r *Repo) FindByPhone(ctx context.Context, phone string) (*Driver, error) {
	const q = `
SELECT
  id, phone, created_at, updated_at, last_online_at, latitude, longitude, push_token,
  registration_step, registration_status, name, driver_type, rating, work_status,
  freelancer_id, company_id, account_status,
  driver_passport_series, driver_passport_number, driver_pinfl, driver_scan_status,
  power_plate_type, power_plate_number, power_tech_series, power_tech_number, power_owner_id, power_owner_name, power_scan_status,
  trailer_plate_type, trailer_plate_number, trailer_tech_series, trailer_tech_number, trailer_owner_id, trailer_owner_name, trailer_scan_status,
  driver_owner, kyc_status
FROM drivers
WHERE phone = $1
LIMIT 1`

	d, err := scanDriver(r.pg.QueryRow(ctx, q, phone))
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *Repo) FindByID(ctx context.Context, id uuid.UUID) (*Driver, error) {
	const q = `
SELECT
  id, phone, created_at, updated_at, last_online_at, latitude, longitude, push_token,
  registration_step, registration_status, name, driver_type, rating, work_status,
  freelancer_id, company_id, account_status,
  driver_passport_series, driver_passport_number, driver_pinfl, driver_scan_status,
  power_plate_type, power_plate_number, power_tech_series, power_tech_number, power_owner_id, power_owner_name, power_scan_status,
  trailer_plate_type, trailer_plate_number, trailer_tech_series, trailer_tech_number, trailer_owner_id, trailer_owner_name, trailer_scan_status,
  driver_owner, kyc_status
FROM drivers
WHERE id = $1
LIMIT 1`

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
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
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
	PowerPlateNumber *string
	PowerTechSeries  *string
	PowerTechNumber  *string
	PowerOwnerID     *string
	PowerOwnerName   *string
	PowerScanStatus  *bool
}

func (r *Repo) UpdatePowerProfile(ctx context.Context, id uuid.UUID, u UpdatePowerProfile) error {
	const q = `
UPDATE drivers
SET power_plate_number = COALESCE($2, power_plate_number),
    power_tech_series = COALESCE($3, power_tech_series),
    power_tech_number = COALESCE($4, power_tech_number),
    power_owner_id = COALESCE($5, power_owner_id),
    power_owner_name = COALESCE($6, power_owner_name),
    power_scan_status = COALESCE($7, power_scan_status),
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, u.PowerPlateNumber, u.PowerTechSeries, u.PowerTechNumber, u.PowerOwnerID, u.PowerOwnerName, u.PowerScanStatus)
	return err
}

type UpdateTrailerProfile struct {
	TrailerPlateNumber *string
	TrailerTechSeries  *string
	TrailerTechNumber  *string
	TrailerOwnerID     *string
	TrailerOwnerName   *string
	TrailerScanStatus  *bool
}

func (r *Repo) UpdateTrailerProfile(ctx context.Context, id uuid.UUID, u UpdateTrailerProfile) error {
	const q = `
UPDATE drivers
SET trailer_plate_number = COALESCE($2, trailer_plate_number),
    trailer_tech_series = COALESCE($3, trailer_tech_series),
    trailer_tech_number = COALESCE($4, trailer_tech_number),
    trailer_owner_id = COALESCE($5, trailer_owner_id),
    trailer_owner_name = COALESCE($6, trailer_owner_name),
    trailer_scan_status = COALESCE($7, trailer_scan_status),
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, u.TrailerPlateNumber, u.TrailerTechSeries, u.TrailerTechNumber, u.TrailerOwnerID, u.TrailerOwnerName, u.TrailerScanStatus)
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
	const q = `
UPDATE drivers
SET driver_type = $2,
    freelancer_id = $3,
    company_id = $4,
    power_plate_type = $5,
    trailer_plate_type = $6,
    registration_step = $7,
    registration_status = $8,
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, driverType, freelancerID, companyID, powerPlateType, trailerPlateType, nextStep, nextStatus)
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
}

func (r *Repo) UpdateKYC(ctx context.Context, id uuid.UUID, u KYCUpdate) error {
	const q = `
UPDATE drivers
SET driver_passport_series = $2,
    driver_passport_number = $3,
    driver_pinfl = $4,
    driver_scan_status = $5,
    power_plate_number = $6,
    power_tech_series = $7,
    power_tech_number = $8,
    power_owner_id = $9,
    power_owner_name = $10,
    power_scan_status = $11,
    trailer_plate_number = $12,
    trailer_tech_series = $13,
    trailer_tech_number = $14,
    trailer_owner_id = $15,
    trailer_owner_name = $16,
    trailer_scan_status = $17,
    driver_owner = $18,
    kyc_status = $19,
    registration_status = $20,
    updated_at = now(),
    last_online_at = now()
WHERE id = $1`

	_, err := r.pg.Exec(ctx, q,
		id,
		u.DriverPassportSeries,
		u.DriverPassportNumber,
		u.DriverPINFL,
		u.DriverScanStatus,
		u.PowerPlateNumber,
		u.PowerTechSeries,
		u.PowerTechNumber,
		u.PowerOwnerID,
		u.PowerOwnerName,
		u.PowerScanStatus,
		u.TrailerPlateNumber,
		u.TrailerTechSeries,
		u.TrailerTechNumber,
		u.TrailerOwnerID,
		u.TrailerOwnerName,
		u.TrailerScanStatus,
		u.DriverOwner,
		u.KYCStatus,
		u.RegistrationStatus,
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

