package goadmin

import (
	"html/template"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/template/types"
)

// DashboardContent returns the panel for the admin index page (GET /admin).
// Cards for all entities from API/Swagger: Companies, Admins, Drivers, Dispatchers, Cargo, Route Points, Payments, Offers, Company Users, App Roles, User Company Roles, Invitations, Audit Log, Chat.
func DashboardContent(ctx *context.Context) (types.Panel, error) {
	prefix := config.Prefix()
	html := `
		<div class="row">
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/companies" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-building"></i></div>
				<div class="admin-dash-card-title">Companies</div>
				<div class="admin-dash-card-desc">Компании (API + Swagger)</div>
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
		<div class="row">
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/cargo" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-cube"></i></div>
				<div class="admin-dash-card-title">Cargo</div>
				<div class="admin-dash-card-desc">Грузы</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/route_points" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-map-marker"></i></div>
				<div class="admin-dash-card-title">Route Points</div>
				<div class="admin-dash-card-desc">Точки маршрута</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/payments" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-money"></i></div>
				<div class="admin-dash-card-title">Payments</div>
				<div class="admin-dash-card-desc">Оплаты по грузам</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/offers" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-handshake-o"></i></div>
				<div class="admin-dash-card-title">Offers</div>
				<div class="admin-dash-card-desc">Офферы перевозчиков</div>
			</a></div>
		</div>
		<div class="row">
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/company_users" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-user"></i></div>
				<div class="admin-dash-card-title">Company Users</div>
				<div class="admin-dash-card-desc">Пользователи компаний (Company TZ)</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/app_roles" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-user-circle"></i></div>
				<div class="admin-dash-card-title">App Roles</div>
				<div class="admin-dash-card-desc">Роли в компаниях</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/user_company_roles" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-link"></i></div>
				<div class="admin-dash-card-title">User Company Roles</div>
				<div class="admin-dash-card-desc">Связь пользователь–компания–роль</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/invitations" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-envelope"></i></div>
				<div class="admin-dash-card-title">Invitations</div>
				<div class="admin-dash-card-desc">Приглашения в компанию</div>
			</a></div>
		</div>
		<div class="row">
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/audit_log" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-history"></i></div>
				<div class="admin-dash-card-title">Audit Log</div>
				<div class="admin-dash-card-desc">Журнал аудита компаний</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/chat_conversations" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-comments"></i></div>
				<div class="admin-dash-card-title">Chat Conversations</div>
				<div class="admin-dash-card-desc">Диалоги чата</div>
			</a></div>
			<div class="col-md-6 col-lg-3"><a href="` + prefix + `/info/chat_messages" class="admin-dash-card">
				<div class="admin-dash-card-icon"><i class="fa fa-comment"></i></div>
				<div class="admin-dash-card-title">Chat Messages</div>
				<div class="admin-dash-card-desc">Сообщения чата</div>
			</a></div>
		</div>
	`
	return types.Panel{
		Content:     template.HTML(html),
		Title:       "Главная",
		Description: "Управление данными (API + Swagger + GoAdmin)",
	}, nil
}


