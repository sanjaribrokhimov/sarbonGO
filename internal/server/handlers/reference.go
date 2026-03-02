package handlers

import (
	"github.com/gin-gonic/gin"

	"sarbonNew/internal/approles"
	"sarbonNew/internal/server/resp"
)

// ReferenceDriversResponse — справочник для раздела Drivers (водители).
type ReferenceDriversResponse struct {
	RegistrationStep   []ItemWithLabel `json:"registration_step"`
	RegistrationStatus []ItemWithLabel `json:"registration_status"`
	DriverType         []ItemWithLabel `json:"driver_type"`
	WorkStatus         []ItemWithLabel `json:"work_status"`
	PowerPlateTypes    []ItemWithLabel `json:"power_plate_types"`
	TrailerPlateTypes  map[string][]ItemWithLabel `json:"trailer_plate_types_by_power"`
}

// ReferenceCargoResponse — справочник для раздела Cargo (грузы).
type ReferenceCargoResponse struct {
	CargoStatus    []ItemWithLabel `json:"cargo_status"`
	RoutePointType []ItemWithLabel `json:"route_point_type"`
	OfferStatus    []ItemWithLabel `json:"offer_status"`
	CreatedByType  []ItemWithLabel `json:"created_by_type"`
	TruckType      []ItemWithLabel `json:"truck_type"`
}

// ReferenceCompanyResponse — справочник для раздела Company.
type ReferenceCompanyResponse struct {
	CompanyType   []ItemWithLabel `json:"company_type"`
	CompanyStatus []ItemWithLabel `json:"company_status"`
	Roles         []RoleRef       `json:"roles"`
}

// ReferenceAdminResponse — справочник для раздела Admin.
type ReferenceAdminResponse struct {
	AdminStatus []ItemWithLabel `json:"admin_status"`
	AdminType   []ItemWithLabel `json:"admin_type"`
}

// ReferenceDispatchersResponse — справочник для раздела Freelance Dispatchers.
type ReferenceDispatchersResponse struct {
	WorkStatus []ItemWithLabel `json:"work_status"`
}

type ItemWithLabel struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type RoleRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

var refDrivers = ReferenceDriversResponse{
	RegistrationStep: []ItemWithLabel{
		{Value: "name-oferta", Label: "Имя и оферта"},
		{Value: "geo-push", Label: "Геолокация и push"},
		{Value: "transport-type", Label: "Тип транспорта"},
	},
	RegistrationStatus: []ItemWithLabel{
		{Value: "start", Label: "Начало"},
		{Value: "basic", Label: "Базовые данные"},
		{Value: "full", Label: "Полная регистрация"},
	},
	DriverType: []ItemWithLabel{
		{Value: "company", Label: "Компания"},
		{Value: "freelancer", Label: "Фрилансер"},
		{Value: "driver", Label: "Водитель"},
	},
	WorkStatus: []ItemWithLabel{
		{Value: "available", Label: "Свободен"},
		{Value: "loaded", Label: "Загружен"},
		{Value: "busy", Label: "Занят"},
	},
	PowerPlateTypes: []ItemWithLabel{
		{Value: "TRUCK", Label: "Грузовик + прицеп"},
		{Value: "TRACTOR", Label: "Тягач + полуприцеп"},
	},
	TrailerPlateTypes: map[string][]ItemWithLabel{
		"TRUCK": {
			{Value: "FLATBED", Label: "Бортовой прицеп"},
			{Value: "TENTED", Label: "Тентованный прицеп"},
			{Value: "BOX", Label: "Фургонный прицеп"},
			{Value: "REEFER", Label: "Рефрижераторный прицеп"},
			{Value: "TANKER", Label: "Прицеп-цистерна"},
			{Value: "TIPPER", Label: "Самосвальный прицеп"},
			{Value: "CAR_CARRIER", Label: "Прицеп-автовоз"},
		},
		"TRACTOR": {
			{Value: "FLATBED", Label: "Бортовой полуприцеп"},
			{Value: "TENTED", Label: "Тентованный полуприцеп"},
			{Value: "BOX", Label: "Фургонный полуприцеп"},
			{Value: "REEFER", Label: "Рефрижераторный полуприцеп"},
			{Value: "TANKER", Label: "Полуприцеп-цистерна"},
			{Value: "LOWBED", Label: "Низкорамный полуприцеп"},
			{Value: "CONTAINER", Label: "Полуприцеп-контейнеровоз"},
		},
	},
}

var refCargo = ReferenceCargoResponse{
	CargoStatus: []ItemWithLabel{
		{Value: "created", Label: "Создан"},
		{Value: "searching", Label: "В поиске перевозчика"},
		{Value: "assigned", Label: "Назначен"},
		{Value: "in_transit", Label: "В пути"},
		{Value: "delivered", Label: "Доставлен"},
		{Value: "cancelled", Label: "Отменён"},
	},
	RoutePointType: []ItemWithLabel{
		{Value: "load", Label: "Погрузка"},
		{Value: "unload", Label: "Выгрузка"},
		{Value: "customs", Label: "Таможня"},
		{Value: "transit", Label: "Транзит"},
	},
	OfferStatus: []ItemWithLabel{
		{Value: "pending", Label: "На рассмотрении"},
		{Value: "accepted", Label: "Принят"},
		{Value: "rejected", Label: "Отклонён"},
	},
	CreatedByType: []ItemWithLabel{
		{Value: "admin", Label: "Админ"},
		{Value: "dispatcher", Label: "Диспетчер"},
		{Value: "company", Label: "Компания"},
	},
	TruckType: []ItemWithLabel{
		{Value: "refrigerator", Label: "Рефрижератор"},
		{Value: "tent", Label: "Тент"},
		{Value: "flatbed", Label: "Борт"},
		{Value: "tanker", Label: "Цистерна"},
		{Value: "other", Label: "Другое"},
	},
}

var refCompany = ReferenceCompanyResponse{
	CompanyType: []ItemWithLabel{
		{Value: "Shipper", Label: "Грузоотправитель"},
		{Value: "Broker", Label: "Брокер"},
		{Value: "Fleet", Label: "Автопарк"},
		{Value: "OwnerOperator", Label: "Владелец-оператор"},
	},
	CompanyStatus: []ItemWithLabel{
		{Value: "active", Label: "Активна"},
		{Value: "inactive", Label: "Неактивна"},
		{Value: "blocked", Label: "Заблокирована"},
		{Value: "pending", Label: "На модерации"},
	},
	Roles: nil, // заполняется из БД
}

var refAdmin = ReferenceAdminResponse{
	AdminStatus: []ItemWithLabel{
		{Value: "active", Label: "Активен"},
		{Value: "inactive", Label: "Неактивен"},
		{Value: "blocked", Label: "Заблокирован"},
	},
	AdminType: []ItemWithLabel{
		{Value: "creator", Label: "Создатель"},
		{Value: "operator", Label: "Оператор"},
	},
}

var refDispatchers = ReferenceDispatchersResponse{
	WorkStatus: []ItemWithLabel{
		{Value: "available", Label: "Доступен"},
		{Value: "busy", Label: "Занят"},
		{Value: "offline", Label: "Не в сети"},
	},
}

// GetReferenceDrivers возвращает справочник для раздела Drivers.
func GetReferenceDrivers(c *gin.Context) {
	resp.OK(c, refDrivers)
}

// GetReferenceCargo возвращает справочник для раздела Cargo.
func GetReferenceCargo(c *gin.Context) {
	resp.OK(c, refCargo)
}

// GetReferenceAdmin возвращает справочник для раздела Admin.
func GetReferenceAdmin(c *gin.Context) {
	resp.OK(c, refAdmin)
}

// GetReferenceDispatchers возвращает справочник для раздела Freelance Dispatchers.
func GetReferenceDispatchers(c *gin.Context) {
	resp.OK(c, refDispatchers)
}

// GetReferenceCompany возвращает справочник для раздела Company (роли из БД).
func GetReferenceCompany(rolesRepo *approles.Repo) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, err := rolesRepo.ListAll(c.Request.Context())
		if err != nil {
			resp.Error(c, 500, "failed to load roles")
			return
		}
		out := refCompany
		out.Roles = make([]RoleRef, 0, len(roles))
		for _, r := range roles {
			desc := ""
			if r.Description != nil {
				desc = *r.Description
			}
			out.Roles = append(out.Roles, RoleRef{ID: r.ID, Name: r.Name, Description: desc})
		}
		resp.OK(c, out)
	}
}
