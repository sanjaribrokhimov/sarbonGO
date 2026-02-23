package goadmin

import (
	"html/template"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/template/types"
)

// DashboardContent returns the panel for the admin index page (GET /admin).
func DashboardContent(ctx *context.Context) (types.Panel, error) {
	prefix := config.Prefix()
	html := `
		<div class="row">
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/companies" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-building"></i></div>
				<div class="admin-dash-card-title">Companies</div>
				<div class="admin-dash-card-desc">Компании</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/admins" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-user-plus"></i></div>
				<div class="admin-dash-card-title">Operator Admins</div>
				<div class="admin-dash-card-desc">Админы (создание компаний)</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/drivers" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-truck"></i></div>
				<div class="admin-dash-card-title">Drivers</div>
				<div class="admin-dash-card-desc">Водители</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/freelance_dispatchers" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-users"></i></div>
				<div class="admin-dash-card-title">Freelance Dispatchers</div>
				<div class="admin-dash-card-desc">Фриланс-диспетчеры</div>
			</a></div>
		</div>
	`
	return types.Panel{
		Content:     template.HTML(html),
		Title:       "Главная",
		Description: "Управление данными",
	}, nil
}


