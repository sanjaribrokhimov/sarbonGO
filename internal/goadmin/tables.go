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
		"companies":              GetCompaniesTable,
		"admins":                 GetOperatorAdminsTable,
		"drivers":                GetDriversTable,
		"freelance_dispatchers":  GetFreelanceDispatchersTable,
		"cargo":                  GetCargoTable,
		"route_points":           GetRoutePointsTable,
		"payments":               GetPaymentsTable,
		"offers":                 GetOffersTable,
		"app_users":              GetAppUsersTable,
		"app_roles":              GetAppRolesTable,
		"user_company_roles":     GetUserCompanyRolesTable,
		"invitations":            GetInvitationsTable,
		"audit_log":              GetAuditLogTable,
		"chat_conversations":     GetChatConversationsTable,
		"chat_messages":          GetChatMessagesTable,
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
	info.AddField("Owner ID", "owner_id", db.Varchar).FieldFilterable()
	info.AddField("Company type", "company_type", db.Varchar).FieldFilterable()
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
	formList.AddField("Owner ID", "owner_id", db.Varchar, form.Text)
	formList.AddField("Company type", "company_type", db.Varchar, form.Text).FieldDefault("Shipper")
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

// GetCargoTable returns the GoAdmin table for cargo (main table).
func GetCargoTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Title", "title", db.Varchar).FieldFilterable()
	info.AddField("Weight", "weight", db.Decimal).FieldFilterable()
	info.AddField("Volume", "volume", db.Decimal)
	info.AddField("Ready enabled", "ready_enabled", db.Boolean)
	info.AddField("Ready at", "ready_at", db.Timestamp)
	info.AddField("Truck type", "truck_type", db.Varchar).FieldFilterable()
	info.AddField("Capacity", "capacity", db.Decimal)
	info.AddField("Temp min", "temp_min", db.Decimal)
	info.AddField("Temp max", "temp_max", db.Decimal)
	info.AddField("ADR enabled", "adr_enabled", db.Boolean)
	info.AddField("ADR class", "adr_class", db.Varchar)
	info.AddField("Status", "status", db.Varchar).FieldFilterable()
	info.AddField("Contact name", "contact_name", db.Varchar)
	info.AddField("Contact phone", "contact_phone", db.Varchar)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.AddField("Updated at", "updated_at", db.Timestamp)
	info.AddField("Deleted at", "deleted_at", db.Timestamp)
	info.AddField("Created by type", "created_by_type", db.Varchar).FieldFilterable()
	info.AddField("Created by ID", "created_by_id", db.Varchar).FieldFilterable()
	info.AddField("Company ID", "company_id", db.Varchar).FieldFilterable()
	info.SetTable("cargo").SetTitle("Cargo").SetDescription("Cargo (main table)")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Title", "title", db.Varchar, form.Text)
	formList.AddField("Weight", "weight", db.Decimal, form.Number)
	formList.AddField("Volume", "volume", db.Decimal, form.Number)
	formList.AddField("Ready enabled", "ready_enabled", db.Boolean, form.Switch).FieldDefault("false")
	formList.AddField("Ready at", "ready_at", db.Timestamp, form.Datetime)
	formList.AddField("Load comment", "load_comment", db.Varchar, form.Text)
	formList.AddField("Truck type", "truck_type", db.Varchar, form.Text)
	formList.AddField("Capacity", "capacity", db.Decimal, form.Number)
	formList.AddField("Temp min", "temp_min", db.Decimal, form.Number)
	formList.AddField("Temp max", "temp_max", db.Decimal, form.Number)
	formList.AddField("ADR enabled", "adr_enabled", db.Boolean, form.Switch).FieldDefault("false")
	formList.AddField("ADR class", "adr_class", db.Varchar, form.Text)
	formList.AddField("Shipment type", "shipment_type", db.Varchar, form.Text)
	formList.AddField("Belts count", "belts_count", db.Int, form.Number)
	formList.AddField("Contact name", "contact_name", db.Varchar, form.Text)
	formList.AddField("Contact phone", "contact_phone", db.Varchar, form.Text)
	formList.AddField("Status", "status", db.Varchar, form.Text).FieldDefault("created")
	formList.AddField("Created by type", "created_by_type", db.Varchar, form.Text)
	formList.AddField("Created by ID", "created_by_id", db.Varchar, form.Text)
	formList.AddField("Company ID", "company_id", db.Varchar, form.Text)
	formList.SetTable("cargo").SetTitle("Cargo").SetDescription("Cargo")
	return
}

// GetRoutePointsTable returns the GoAdmin table for route_points.
func GetRoutePointsTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Cargo ID", "cargo_id", db.Varchar).FieldFilterable()
	info.AddField("Type", "type", db.Varchar).FieldFilterable()
	info.AddField("Address", "address", db.Varchar)
	info.AddField("Lat", "lat", db.Decimal)
	info.AddField("Lng", "lng", db.Decimal)
	info.AddField("Comment", "comment", db.Varchar)
	info.AddField("Point order", "point_order", db.Int)
	info.AddField("Is main load", "is_main_load", db.Boolean)
	info.AddField("Is main unload", "is_main_unload", db.Boolean)
	info.SetTable("route_points").SetTitle("Route Points").SetDescription("Route points (load/unload/customs/transit)")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Cargo ID", "cargo_id", db.Varchar, form.Text)
	formList.AddField("Type", "type", db.Varchar, form.Text)
	formList.AddField("Address", "address", db.Varchar, form.Text)
	formList.AddField("Lat", "lat", db.Decimal, form.Number)
	formList.AddField("Lng", "lng", db.Decimal, form.Number)
	formList.AddField("Comment", "comment", db.Varchar, form.Text)
	formList.AddField("Point order", "point_order", db.Int, form.Number)
	formList.AddField("Is main load", "is_main_load", db.Boolean, form.Switch).FieldDefault("false")
	formList.AddField("Is main unload", "is_main_unload", db.Boolean, form.Switch).FieldDefault("false")
	formList.SetTable("route_points").SetTitle("Route Points").SetDescription("Route points")
	return
}

// GetPaymentsTable returns the GoAdmin table for payments.
func GetPaymentsTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Cargo ID", "cargo_id", db.Varchar).FieldFilterable()
	info.AddField("Is negotiable", "is_negotiable", db.Boolean)
	info.AddField("Price request", "price_request", db.Boolean)
	info.AddField("Total amount", "total_amount", db.Decimal)
	info.AddField("Total currency", "total_currency", db.Varchar)
	info.AddField("With prepayment", "with_prepayment", db.Boolean)
	info.AddField("Without prepayment", "without_prepayment", db.Boolean)
	info.AddField("Prepayment amount", "prepayment_amount", db.Decimal)
	info.AddField("Prepayment currency", "prepayment_currency", db.Varchar)
	info.AddField("Remaining amount", "remaining_amount", db.Decimal)
	info.AddField("Remaining currency", "remaining_currency", db.Varchar)
	info.SetTable("payments").SetTitle("Payments").SetDescription("Payments (1:1 with cargo)")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Cargo ID", "cargo_id", db.Varchar, form.Text)
	formList.AddField("Is negotiable", "is_negotiable", db.Boolean, form.Switch).FieldDefault("false")
	formList.AddField("Price request", "price_request", db.Boolean, form.Switch).FieldDefault("false")
	formList.AddField("Total amount", "total_amount", db.Decimal, form.Number)
	formList.AddField("Total currency", "total_currency", db.Varchar, form.Text)
	formList.AddField("With prepayment", "with_prepayment", db.Boolean, form.Switch).FieldDefault("false")
	formList.AddField("Without prepayment", "without_prepayment", db.Boolean, form.Switch).FieldDefault("true")
	formList.AddField("Prepayment amount", "prepayment_amount", db.Decimal, form.Number)
	formList.AddField("Prepayment currency", "prepayment_currency", db.Varchar, form.Text)
	formList.AddField("Remaining amount", "remaining_amount", db.Decimal, form.Number)
	formList.AddField("Remaining currency", "remaining_currency", db.Varchar, form.Text)
	formList.SetTable("payments").SetTitle("Payments").SetDescription("Payments")
	return
}

// GetOffersTable returns the GoAdmin table for offers.
func GetOffersTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Cargo ID", "cargo_id", db.Varchar).FieldFilterable()
	info.AddField("Carrier ID", "carrier_id", db.Varchar)
	info.AddField("Price", "price", db.Decimal)
	info.AddField("Currency", "currency", db.Varchar)
	info.AddField("Comment", "comment", db.Varchar)
	info.AddField("Status", "status", db.Varchar).FieldFilterable()
	info.AddField("Created at", "created_at", db.Timestamp)
	info.SetTable("offers").SetTitle("Offers").SetDescription("Carrier offers for cargo")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Cargo ID", "cargo_id", db.Varchar, form.Text)
	formList.AddField("Carrier ID", "carrier_id", db.Varchar, form.Text)
	formList.AddField("Price", "price", db.Decimal, form.Number)
	formList.AddField("Currency", "currency", db.Varchar, form.Text)
	formList.AddField("Comment", "comment", db.Varchar, form.Text)
	formList.AddField("Status", "status", db.Varchar, form.Text).FieldDefault("pending")
	formList.SetTable("offers").SetTitle("Offers").SetDescription("Offers")
	return
}

// GetAppUsersTable — пользователи приложения (Company TZ: регистрация, владельцы компаний).
func GetAppUsersTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Email", "email", db.Varchar).FieldFilterable()
	info.AddField("Phone", "phone", db.Varchar).FieldFilterable()
	info.AddField("First name", "first_name", db.Varchar)
	info.AddField("Last name", "last_name", db.Varchar)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.AddField("Updated at", "updated_at", db.Timestamp)
	info.SetTable("app_users").SetTitle("App Users").SetDescription("Users (register/login, company owners)")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Email", "email", db.Varchar, form.Text)
	formList.AddField("Phone", "phone", db.Varchar, form.Text)
	formList.AddField("Password hash", "password_hash", db.Varchar, form.Password)
	formList.AddField("First name", "first_name", db.Varchar, form.Text)
	formList.AddField("Last name", "last_name", db.Varchar, form.Text)
	formList.SetTable("app_users").SetTitle("App Users").SetDescription("App Users")
	return
}

// GetAppRolesTable — роли в компаниях (Owner, CEO, Dispatcher и т.д.).
func GetAppRolesTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Name", "name", db.Varchar).FieldFilterable()
	info.AddField("Description", "description", db.Varchar)
	info.SetTable("app_roles").SetTitle("App Roles").SetDescription("Company roles (Owner, CEO, Dispatcher...)")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Name", "name", db.Varchar, form.Text)
	formList.AddField("Description", "description", db.Varchar, form.Text)
	formList.SetTable("app_roles").SetTitle("App Roles").SetDescription("App Roles")
	return
}

// GetUserCompanyRolesTable — связь пользователь–компания–роль.
func GetUserCompanyRolesTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("user_id", db.Varchar))
	info := t.GetInfo()
	info.AddField("User ID", "user_id", db.Varchar).FieldSortable().FieldFilterable()
	info.AddField("Company ID", "company_id", db.Varchar).FieldFilterable()
	info.AddField("Role ID", "role_id", db.Varchar).FieldFilterable()
	info.AddField("Assigned by", "assigned_by", db.Varchar)
	info.AddField("Assigned at", "assigned_at", db.Timestamp)
	info.SetTable("user_company_roles").SetTitle("User Company Roles").SetDescription("User roles per company")

	formList := t.GetForm()
	formList.AddField("User ID", "user_id", db.Varchar, form.Text)
	formList.AddField("Company ID", "company_id", db.Varchar, form.Text)
	formList.AddField("Role ID", "role_id", db.Varchar, form.Text)
	formList.AddField("Assigned by", "assigned_by", db.Varchar, form.Text)
	formList.AddField("Assigned at", "assigned_at", db.Timestamp, form.Datetime)
	formList.SetTable("user_company_roles").SetTitle("User Company Roles").SetDescription("User Company Roles")
	return
}

// GetInvitationsTable — приглашения в компанию.
func GetInvitationsTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Token", "token", db.Varchar).FieldFilterable()
	info.AddField("Company ID", "company_id", db.Varchar).FieldFilterable()
	info.AddField("Role ID", "role_id", db.Varchar)
	info.AddField("Email", "email", db.Varchar).FieldFilterable()
	info.AddField("Invited by", "invited_by", db.Varchar)
	info.AddField("Expires at", "expires_at", db.Timestamp)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.SetTable("invitations").SetTitle("Invitations").SetDescription("Company invitations")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Token", "token", db.Varchar, form.Text)
	formList.AddField("Company ID", "company_id", db.Varchar, form.Text)
	formList.AddField("Role ID", "role_id", db.Varchar, form.Text)
	formList.AddField("Email", "email", db.Varchar, form.Text)
	formList.AddField("Invited by", "invited_by", db.Varchar, form.Text)
	formList.AddField("Expires at", "expires_at", db.Timestamp, form.Datetime)
	formList.SetTable("invitations").SetTitle("Invitations").SetDescription("Invitations")
	return
}

// GetAuditLogTable — журнал аудита (действия в компаниях).
func GetAuditLogTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("User ID", "user_id", db.Varchar).FieldFilterable()
	info.AddField("Company ID", "company_id", db.Varchar).FieldFilterable()
	info.AddField("Action", "action", db.Varchar).FieldFilterable()
	info.AddField("Entity type", "entity_type", db.Varchar)
	info.AddField("Entity ID", "entity_id", db.Varchar)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.SetTable("audit_log").SetTitle("Audit Log").SetDescription("Company audit log")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("User ID", "user_id", db.Varchar, form.Text)
	formList.AddField("Company ID", "company_id", db.Varchar, form.Text)
	formList.AddField("Action", "action", db.Varchar, form.Text)
	formList.AddField("Entity type", "entity_type", db.Varchar, form.Text)
	formList.AddField("Entity ID", "entity_id", db.Varchar, form.Text)
	formList.SetTable("audit_log").SetTitle("Audit Log").SetDescription("Audit Log")
	return
}

// GetChatConversationsTable — диалоги чата.
func GetChatConversationsTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("User A ID", "user_a_id", db.Varchar).FieldFilterable()
	info.AddField("User B ID", "user_b_id", db.Varchar).FieldFilterable()
	info.AddField("Created at", "created_at", db.Timestamp)
	info.SetTable("chat_conversations").SetTitle("Chat Conversations").SetDescription("Chat conversations")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("User A ID", "user_a_id", db.Varchar, form.Text)
	formList.AddField("User B ID", "user_b_id", db.Varchar, form.Text)
	formList.SetTable("chat_conversations").SetTitle("Chat Conversations").SetDescription("Chat Conversations")
	return
}

// GetChatMessagesTable — сообщения чата.
func GetChatMessagesTable(ctx *context.Context) (t table.Table) {
	t = table.NewDefaultTable(ctx, table.DefaultConfigWithDriver(db.DriverPostgresql).
		SetPrimaryKey("id", db.Varchar))
	info := t.GetInfo()
	info.AddField("ID", "id", db.Varchar).FieldSortable()
	info.AddField("Conversation ID", "conversation_id", db.Varchar).FieldFilterable()
	info.AddField("Sender ID", "sender_id", db.Varchar).FieldFilterable()
	info.AddField("Body", "body", db.Varchar)
	info.AddField("Created at", "created_at", db.Timestamp)
	info.AddField("Updated at", "updated_at", db.Timestamp)
	info.AddField("Deleted at", "deleted_at", db.Timestamp)
	info.SetTable("chat_messages").SetTitle("Chat Messages").SetDescription("Chat messages")

	formList := t.GetForm()
	formList.AddField("ID", "id", db.Varchar, form.Default).FieldDisplayButCanNotEditWhenUpdate().FieldDisableWhenCreate()
	formList.AddField("Conversation ID", "conversation_id", db.Varchar, form.Text)
	formList.AddField("Sender ID", "sender_id", db.Varchar, form.Text)
	formList.AddField("Body", "body", db.Varchar, form.Text)
	formList.AddField("Created at", "created_at", db.Timestamp, form.Datetime)
	formList.AddField("Updated at", "updated_at", db.Timestamp, form.Datetime)
	formList.AddField("Deleted at", "deleted_at", db.Timestamp, form.Datetime)
	formList.SetTable("chat_messages").SetTitle("Chat Messages").SetDescription("Chat Messages")
	return
}
