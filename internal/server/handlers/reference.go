package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"sarbonNew/internal/approles"
	"sarbonNew/internal/reference"
	"sarbonNew/internal/server/resp"
)

// ReferenceDriversResponse — справочник для раздела Drivers (водители). Все value в верхнем регистре.
type ReferenceDriversResponse struct {
	RegistrationStep   []ItemWithLabel `json:"registration_step"`
	RegistrationStatus []ItemWithLabel `json:"registration_status"`
	DriverType         []ItemWithLabel `json:"driver_type"`
	WorkStatus         []ItemWithLabel `json:"work_status"`
	PowerPlateTypes    []ItemWithLabel `json:"power_plate_types"`
	TrailerPlateTypes  map[string][]ItemWithLabel `json:"trailer_plate_types_by_power"`
}

// ReferenceCargoResponse — справочник для раздела Cargo (грузы). Все value в верхнем регистре.
type ReferenceCargoResponse struct {
	CargoStatus    []ItemWithLabel `json:"cargo_status"`
	RoutePointType []ItemWithLabel `json:"route_point_type"`
	OfferStatus    []ItemWithLabel `json:"offer_status"`
	CreatedByType  []ItemWithLabel `json:"created_by_type"`
	TruckType      []ItemWithLabel `json:"truck_type"`
	TripStatus     []ItemWithLabel `json:"trip_status"` // статусы рейса
}

// ReferenceCompanyResponse — справочник для раздела Company. Все value в верхнем регистре.
type ReferenceCompanyResponse struct {
	CompanyType      []ItemWithLabel `json:"company_type"`      // только SHIPPER, CARRIER, BROKER
	CompanyStatus    []ItemWithLabel `json:"company_status"`
	CompanyUserRoles []ItemWithLabel `json:"company_user_roles"` // роли пользователей компании: OWNER, CEO, TOP_MANAGER, TOP_DISPATCHER, DISPATCHER, MANAGER
	Roles             []RoleRef       `json:"roles"`             // из БД (id, name, description) для приглашений
}

// ReferenceAdminResponse — справочник для раздела Admin. Все value в верхнем регистре.
type ReferenceAdminResponse struct {
	AdminStatus []ItemWithLabel `json:"admin_status"`
	AdminType   []ItemWithLabel `json:"admin_type"`
}

// ReferenceDispatchersResponse — справочник для раздела Freelance Dispatchers. Все value в верхнем регистре.
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
		{Value: "NAME-OFERTA", Label: "Имя и оферта"},
		{Value: "GEO-PUSH", Label: "Геолокация и push"},
		{Value: "TRANSPORT-TYPE", Label: "Тип транспорта"},
		{Value: "COMPLETED", Label: "Регистрация завершена"},
	},
	RegistrationStatus: []ItemWithLabel{
		{Value: "START", Label: "Начало"},
		{Value: "BASIC", Label: "Базовые данные"},
		{Value: "FULL", Label: "Полная регистрация"},
	},
	DriverType: []ItemWithLabel{
		{Value: "COMPANY", Label: "Компания"},
		{Value: "FREELANCER", Label: "Фрилансер"},
		{Value: "DRIVER", Label: "Водитель"},
	},
	WorkStatus: []ItemWithLabel{
		{Value: "AVAILABLE", Label: "Свободен"},
		{Value: "LOADED", Label: "Загружен"},
		{Value: "BUSY", Label: "Занят"},
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
		{Value: "CREATED", Label: "Создан"},
		{Value: "SEARCHING", Label: "В поиске перевозчика"},
		{Value: "ASSIGNED", Label: "Назначен"},
		{Value: "IN_TRANSIT", Label: "В пути"},
		{Value: "DELIVERED", Label: "Доставлен"},
		{Value: "CANCELLED", Label: "Отменён"},
	},
	RoutePointType: []ItemWithLabel{
		{Value: "LOAD", Label: "Погрузка"},
		{Value: "UNLOAD", Label: "Выгрузка"},
		{Value: "CUSTOMS", Label: "Таможня"},
		{Value: "TRANSIT", Label: "Транзит"},
	},
	OfferStatus: []ItemWithLabel{
		{Value: "PENDING", Label: "На рассмотрении"},
		{Value: "ACCEPTED", Label: "Принят"},
		{Value: "REJECTED", Label: "Отклонён"},
	},
	CreatedByType: []ItemWithLabel{
		{Value: "ADMIN", Label: "Админ"},
		{Value: "DISPATCHER", Label: "Диспетчер"},
		{Value: "COMPANY", Label: "Компания"},
	},
	TruckType: []ItemWithLabel{
		{Value: "REFRIGERATOR", Label: "Рефрижератор"},
		{Value: "TENT", Label: "Тент"},
		{Value: "FLATBED", Label: "Борт"},
		{Value: "TANKER", Label: "Цистерна"},
		{Value: "OTHER", Label: "Другое"},
	},
	TripStatus: []ItemWithLabel{
		{Value: "PENDING_DRIVER", Label: "Ожидание водителя"},
		{Value: "ASSIGNED", Label: "Назначен"},
		{Value: "LOADING", Label: "Погрузка"},
		{Value: "EN_ROUTE", Label: "В пути"},
		{Value: "UNLOADING", Label: "Выгрузка"},
		{Value: "COMPLETED", Label: "Завершён"},
		{Value: "CANCELLED", Label: "Отменён"},
	},
}

// Допустимые роли пользователей компании (company_users.role) — только эти 6, в верхнем регистре.
var refCompanyUserRoles = []ItemWithLabel{
	{Value: "OWNER", Label: "Владелец"},
	{Value: "CEO", Label: "Директор"},
	{Value: "TOP_MANAGER", Label: "Старший менеджер"},
	{Value: "TOP_DISPATCHER", Label: "Старший диспетчер"},
	{Value: "DISPATCHER", Label: "Диспетчер"},
	{Value: "MANAGER", Label: "Менеджер"},
}

// Допустимые типы компании — только SHIPPER, CARRIER, BROKER (верхний регистр).
var refCompanyType = []ItemWithLabel{
	{Value: "SHIPPER", Label: "Грузоотправитель"},
	{Value: "CARRIER", Label: "Перевозчик"},
	{Value: "BROKER", Label: "Брокер"},
}

var refCompany = ReferenceCompanyResponse{
	CompanyType:      refCompanyType,
	CompanyStatus: []ItemWithLabel{
		{Value: "ACTIVE", Label: "Активна"},
		{Value: "INACTIVE", Label: "Неактивна"},
		{Value: "BLOCKED", Label: "Заблокирована"},
		{Value: "PENDING", Label: "На модерации"},
	},
	CompanyUserRoles: refCompanyUserRoles,
	Roles:             nil, // заполняется из БД
}

var refAdmin = ReferenceAdminResponse{
	AdminStatus: []ItemWithLabel{
		{Value: "ACTIVE", Label: "Активен"},
		{Value: "INACTIVE", Label: "Неактивен"},
		{Value: "BLOCKED", Label: "Заблокирован"},
	},
	AdminType: []ItemWithLabel{
		{Value: "CREATOR", Label: "Создатель"},
		{Value: "OPERATOR", Label: "Оператор"},
	},
}

var refDispatchers = ReferenceDispatchersResponse{
	WorkStatus: []ItemWithLabel{
		{Value: "AVAILABLE", Label: "Доступен"},
		{Value: "BUSY", Label: "Занят"},
		{Value: "OFFLINE", Label: "Не в сети"},
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
			out.Roles = append(out.Roles, RoleRef{ID: r.ID, Name: strings.ToUpper(r.Name), Description: desc})
		}
		resp.OK(c, out)
	}
}

// CityRef — элемент справочника городов (код TAS, SAM, DXB и т.д.).
type CityRef struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	NameRu      string   `json:"name_ru"`
	NameEn      *string  `json:"name_en,omitempty"`
	CountryCode string   `json:"country_code"`
	Lat         *float64 `json:"lat,omitempty"`
	Lng         *float64 `json:"lng,omitempty"`
}

// GetReferenceCities возвращает справочник городов мира из встроенного датасета (in-memory, быстрый API).
// Query: country_code — фильтр по стране (UZ, AE, RU и т.д.). Данные: ~150k городов (lutangar/cities.json).
func GetReferenceCities() gin.HandlerFunc {
	return func(c *gin.Context) {
		countryCode := strings.TrimSpace(c.Query("country_code"))
		list, err := reference.CitiesByCountry(countryCode)
		if err != nil {
			resp.Error(c, 500, "failed to load cities")
			return
		}
		// Convert to handler response shape (reference.CityRef already matches)
		items := make([]CityRef, len(list))
		for i := range list {
			items[i] = CityRef{
				ID:          list[i].ID,
				Code:        list[i].Code,
				NameRu:      list[i].NameRu,
				NameEn:      list[i].NameEn,
				CountryCode: list[i].CountryCode,
				Lat:         list[i].Lat,
				Lng:         list[i].Lng,
			}
		}
		resp.OK(c, gin.H{"items": items})
	}
}
