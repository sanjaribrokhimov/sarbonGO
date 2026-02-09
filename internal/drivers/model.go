package drivers

import "time"

// Mirrors DB columns from the single table `drivers`.
type Driver struct {
	ID string `json:"id"`

	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	LastOnlineAt *time.Time `json:"last_online_at"`
	Latitude     *float64   `json:"latitude"`
	Longitude    *float64   `json:"longitude"`
	PushToken    *string    `json:"push_token"`

	RegistrationStep   *string `json:"registration_step"`
	RegistrationStatus *string `json:"registration_status"`

	Name *string `json:"name"`

	DriverType *string  `json:"driver_type"`
	Rating     *float64 `json:"rating"`
	WorkStatus *string  `json:"work_status"`

	FreelancerID *string `json:"freelancer_id"`
	CompanyID    *string `json:"company_id"`

	AccountStatus *string `json:"account_status"`

	DriverPassportSeries *string `json:"driver_passport_series"`
	DriverPassportNumber *string `json:"driver_passport_number"`
	DriverPINFL          *string `json:"driver_pinfl"`
	DriverScanStatus     *bool   `json:"driver_scan_status"`

	PowerPlateType     *string `json:"power_plate_type"`
	PowerPlateNumber   *string `json:"power_plate_number"`
	PowerTechSeries    *string `json:"power_tech_series"`
	PowerTechNumber    *string `json:"power_tech_number"`
	PowerOwnerID       *string `json:"power_owner_id"`
	PowerOwnerName     *string `json:"power_owner_name"`
	PowerScanStatus    *bool   `json:"power_scan_status"`

	TrailerPlateType     *string `json:"trailer_plate_type"`
	TrailerPlateNumber   *string `json:"trailer_plate_number"`
	TrailerTechSeries    *string `json:"trailer_tech_series"`
	TrailerTechNumber    *string `json:"trailer_tech_number"`
	TrailerOwnerID       *string `json:"trailer_owner_id"`
	TrailerOwnerName     *string `json:"trailer_owner_name"`
	TrailerScanStatus    *bool   `json:"trailer_scan_status"`

	DriverOwner *bool   `json:"driver_owner"`
	KYCStatus     *string `json:"kyc_status"`
}

