package goadmin

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	editType "github.com/GoAdminGroup/go-admin/template/types/table"
)

// tableGenerators returns the map of table name -> generator for GoAdmin.
func tableGenerators() map[string]table.Generator {
	return map[string]table.Generator{
		"companies":             GetCompaniesTable,
		"admins":                GetOperatorAdminsTable,
		"drivers":               GetDriversTable,
		"freelance_dispatchers": GetFreelanceDispatchersTable,
	}
}

// GetCompaniesTable returns the GoAdmin table descriptor for companies (full CRUD, all fields).
func GetCompaniesTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Name", "name", db.Varchar).FieldFilterable()
	info.AddField("INN", "inn", db.Varchar).FieldFilterable()
	info.AddField("Address", "address", db.Varchar)
	info.AddField("Phone", "phone", db.Varchar)
	info.AddField("Email", "email", db.Varchar)
	info.AddField("Website", "website", db.Varchar)
	info.AddField("License", "license_number", db.Varchar)
	info.AddField("Status", "status", db.Varchar).FieldFilterable()
	info.AddField("Max vehicles", "max_vehicles", db.Int)
	info.AddField("Max drivers", "max_drivers", db.Int)
	info.AddField("Max cargo", "max_cargo", db.Int)
	info.AddField("Max dispatchers", "max_dispatchers", db.Int)
	info.AddField("Max managers", "max_managers", db.Int)
	info.AddField("Max top dispatchers", "max_top_dispatchers", db.Int)
	info.AddField("Max top managers", "max_top_managers", db.Int)
	info.AddField("Rating", "rating", db.Decimal)
	info.AddField("Completed orders", "completed_orders", db.Int)
	info.AddField("Cancelled orders", "cancelled_orders", db.Int)
	info.AddField("Total revenue", "total_revenue", db.Decimal)
	info.AddField("Created by", "created_by", db.Varchar)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.AddField("Updated at", "updated_at", db.Timestamp)
	info.AddField("Deleted at", "deleted_at", db.Timestamp)
	info.SetTable("companies").SetTitle("Companies").SetDescription("Companies")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Name", "name", db.Varchar, form.Text)
	formList.AddField("INN", "inn", db.Varchar, form.Text)
	formList.AddField("Address", "address", db.Varchar, form.Text)
	formList.AddField("Phone", "phone", db.Varchar, form.Text)
	formList.AddField("Email", "email", db.Varchar, form.Text)
	formList.AddField("Website", "website", db.Varchar, form.Text)
	formList.AddField("License number", "license_number", db.Varchar, form.Text)
	formList.AddField("Status", "status", db.Varchar, form.Text).FieldDefault("pending")
	formList.AddField("Max vehicles", "max_vehicles", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Max drivers", "max_drivers", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Max cargo", "max_cargo", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Max dispatchers", "max_dispatchers", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Max managers", "max_managers", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Max top dispatchers", "max_top_dispatchers", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Max top managers", "max_top_managers", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Rating", "rating", db.Decimal, form.Text)
	formList.AddField("Completed orders", "completed_orders", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Cancelled orders", "cancelled_orders", db.Int, form.Number).FieldDefault("0")
	formList.AddField("Total revenue", "total_revenue", db.Decimal, form.Text)
	// created_by — UUID; не в форме, чтобы не отправлять "" в PostgreSQL
	formList.SetTable("companies").SetTitle("Companies").SetDescription("Companies")
	return
}

// GetOperatorAdminsTable returns the GoAdmin table for operator admins (they can create companies via API).
func GetOperatorAdminsTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Login", "login", db.Varchar).FieldFilterable()
	info.AddField("Name", "name", db.Varchar).FieldFilterable()
	info.AddField("Status", "status", db.Varchar).FieldFilterable().FieldEditAble(editType.Text)
	info.AddField("Type", "type", db.Varchar).FieldFilterable()
	info.SetTable("admins").SetTitle("Operator Admins").SetDescription("Admins who can create companies (login via API)")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Login", "login", db.Varchar, form.Text)
	formList.AddField("Password", "password", db.Varchar, form.Password)
	formList.AddField("Name", "name", db.Varchar, form.Text)
	formList.AddField("Status", "status", db.Varchar, form.Text).FieldDefault("active")
	formList.AddField("Type", "type", db.Varchar, form.Text).FieldDefault("creator")
	formList.SetTable("admins").SetTitle("Operator Admins").SetDescription("Admins who can create companies")
	return
}

// GetDriversTable returns the GoAdmin table for drivers (full CRUD, all fields).
func GetDriversTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Phone", "phone", db.Varchar).FieldFilterable()
	info.AddField("Name", "name", db.Varchar).FieldFilterable()
	info.AddField("Driver type", "driver_type", db.Varchar).FieldFilterable()
	info.AddField("Work status", "work_status", db.Varchar)
	info.AddField("Account status", "account_status", db.Varchar)
	info.AddField("Registration step", "registration_step", db.Varchar)
	info.AddField("Registration status", "registration_status", db.Varchar)
	info.AddField("Rating", "rating", db.Decimal)
	info.AddField("Company ID", "company_id", db.Varchar)
	info.AddField("Freelancer ID", "freelancer_id", db.Varchar)
	info.AddField("KYC status", "kyc_status", db.Varchar)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.AddField("Updated at", "updated_at", db.Timestamp)
	info.AddField("Last online", "last_online_at", db.Timestamp)
	info.SetTable("drivers").SetTitle("Drivers").SetDescription("Drivers")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Phone", "phone", db.Varchar, form.Text)
	formList.AddField("Name", "name", db.Varchar, form.Text)
	formList.AddField("Driver type", "driver_type", db.Varchar, form.Text)
	formList.AddField("Work status", "work_status", db.Varchar, form.Text)
	formList.AddField("Account status", "account_status", db.Varchar, form.Text)
	formList.AddField("Registration step", "registration_step", db.Varchar, form.Text)
	formList.AddField("Registration status", "registration_status", db.Varchar, form.Text)
	formList.AddField("Rating", "rating", db.Decimal, form.Text)
	// company_id, freelancer_id — UUID; не добавляем в форму, иначе пустая строка "" даёт ошибку PostgreSQL
	formList.AddField("KYC status", "kyc_status", db.Varchar, form.Text)
	formList.AddField("Driver passport series", "driver_passport_series", db.Varchar, form.Text)
	formList.AddField("Driver passport number", "driver_passport_number", db.Varchar, form.Text)
	formList.AddField("Driver PINFL", "driver_pinfl", db.Varchar, form.Text)
	formList.AddField("Power plate type", "power_plate_type", db.Varchar, form.Text)
	formList.AddField("Power plate number", "power_plate_number", db.Varchar, form.Text)
	formList.AddField("Trailer plate type", "trailer_plate_type", db.Varchar, form.Text)
	formList.AddField("Trailer plate number", "trailer_plate_number", db.Varchar, form.Text)
	formList.SetTable("drivers").SetTitle("Drivers").SetDescription("Drivers")
	return
}

// GetFreelanceDispatchersTable returns the GoAdmin table for freelance dispatchers (full CRUD, all fields).
func GetFreelanceDispatchersTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Name", "name", db.Varchar).FieldFilterable()
	info.AddField("Phone", "phone", db.Varchar).FieldFilterable()
	info.AddField("Work status", "work_status", db.Varchar)
	info.AddField("Account status", "account_status", db.Varchar)
	info.AddField("Rating", "rating", db.Decimal)
	info.AddField("Cargo ID", "cargo_id", db.Varchar)
	info.AddField("Driver ID", "driver_id", db.Varchar)
	info.AddField("Passport series", "passport_series", db.Varchar)
	info.AddField("Passport number", "passport_number", db.Varchar)
	info.AddField("PINFL", "pinfl", db.Varchar)
	info.AddField("Photo path", "photo_path", db.Varchar)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.AddField("Updated at", "updated_at", db.Timestamp)
	info.AddField("Deleted at", "deleted_at", db.Timestamp)
	info.SetTable("freelance_dispatchers").SetTitle("Freelance Dispatchers").SetDescription("Freelance dispatchers")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Name", "name", db.Varchar, form.Text)
	formList.AddField("Phone", "phone", db.Varchar, form.Text)
	formList.AddField("Password", "password", db.Varchar, form.Password)
	formList.AddField("Work status", "work_status", db.Varchar, form.Text)
	formList.AddField("Account status", "account_status", db.Varchar, form.Text)
	formList.AddField("Rating", "rating", db.Decimal, form.Text)
	// cargo_id, driver_id — UUID; не в форме, чтобы не отправлять "" в PostgreSQL
	formList.AddField("Passport series", "passport_series", db.Varchar, form.Text)
	formList.AddField("Passport number", "passport_number", db.Varchar, form.Text)
	formList.AddField("PINFL", "pinfl", db.Varchar, form.Text)
	formList.AddField("Photo path", "photo_path", db.Varchar, form.Text)
	formList.SetTable("freelance_dispatchers").SetTitle("Freelance Dispatchers").SetDescription("Freelance dispatchers")
	return
}
