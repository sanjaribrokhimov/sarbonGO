package handlers

import (
	"net/http"
	"sort"
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

// ReferenceCargoResponse — справочник для раздела Cargo (грузы). Все value в верхнем регистре (кроме payment/loading — там как в API).
// cargo_status: первый статус — created (при создании груза); остальные с описанием.
type ReferenceCargoResponse struct {
	CargoStatus     []ItemWithLabelAndDescription `json:"cargo_status"`
	RoutePointType  []ItemWithLabel               `json:"route_point_type"`
	OfferStatus     []ItemWithLabel               `json:"offer_status"`
	CreatedByType   []ItemWithLabel               `json:"created_by_type"`
	TruckType       []ItemWithLabel               `json:"truck_type"`
	TripStatus      []ItemWithLabel               `json:"trip_status"`
	ShipmentType    []ItemWithLabel               `json:"shipment_type"`
	Currency        []ItemWithLabel               `json:"currency"`
	PrepaymentType  []ItemWithLabel               `json:"prepayment_type"`
	RemainingType   []ItemWithLabel               `json:"remaining_type"`
	LoadingType     []ItemWithLabel               `json:"loading_type"`
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

// ItemWithLabelAndDescription — элемент справочника с пояснением (например статусы груза).
type ItemWithLabelAndDescription struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type RoleRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Label       string `json:"label"` // localized by X-Language
	Description string `json:"description,omitempty"`
}

func refItemsToItemWithLabel(items []reference.RefItem) []ItemWithLabel {
	out := make([]ItemWithLabel, 0, len(items))
	for _, i := range items {
		out = append(out, ItemWithLabel{Value: i.Value, Label: i.Label})
	}
	return out
}

// refItemsToItemWithLabelLocalized: value in response = uppercase, label = RefLabel(section, value, lang).
func refItemsToItemWithLabelLocalized(items []reference.RefItem, section, lang string) []ItemWithLabel {
	out := make([]ItemWithLabel, 0, len(items))
	for _, i := range items {
		val := strings.TrimSpace(i.Value)
		valueUpper := strings.ToUpper(val)
		label := reference.RefLabel(section, val, lang)
		out = append(out, ItemWithLabel{Value: valueUpper, Label: label})
	}
	return out
}

// refLang reads X-Language (ru, uz, en, tr, zh). Returns "en" if invalid or missing.
func refLang(c *gin.Context) string {
	lang := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Language")))
	switch lang {
	case "ru", "uz", "en", "tr", "zh":
		return lang
	}
	return "en"
}

// GetReferenceDrivers возвращает справочник для раздела Drivers. value — верхний регистр, label — по X-Language.
func GetReferenceDrivers(c *gin.Context) {
	lang := refLang(c)
	out := ReferenceDriversResponse{
		RegistrationStep: []ItemWithLabel{
			{Value: "NAME-OFERTA", Label: reference.RefLabel("drivers.registration_step", "NAME-OFERTA", lang)},
			{Value: "GEO-PUSH", Label: reference.RefLabel("drivers.registration_step", "GEO-PUSH", lang)},
			{Value: "TRANSPORT-TYPE", Label: reference.RefLabel("drivers.registration_step", "TRANSPORT-TYPE", lang)},
			{Value: "COMPLETED", Label: reference.RefLabel("drivers.registration_step", "COMPLETED", lang)},
		},
		RegistrationStatus: []ItemWithLabel{
			{Value: "START", Label: reference.RefLabel("drivers.registration_status", "START", lang)},
			{Value: "BASIC", Label: reference.RefLabel("drivers.registration_status", "BASIC", lang)},
			{Value: "FULL", Label: reference.RefLabel("drivers.registration_status", "FULL", lang)},
		},
		DriverType: []ItemWithLabel{
			{Value: "COMPANY", Label: reference.RefLabel("drivers.driver_type", "COMPANY", lang)},
			{Value: "FREELANCER", Label: reference.RefLabel("drivers.driver_type", "FREELANCER", lang)},
			{Value: "DRIVER", Label: reference.RefLabel("drivers.driver_type", "DRIVER", lang)},
		},
		WorkStatus: []ItemWithLabel{
			{Value: "AVAILABLE", Label: reference.RefLabel("drivers.work_status", "AVAILABLE", lang)},
			{Value: "LOADED", Label: reference.RefLabel("drivers.work_status", "LOADED", lang)},
			{Value: "BUSY", Label: reference.RefLabel("drivers.work_status", "BUSY", lang)},
		},
		PowerPlateTypes: []ItemWithLabel{
			{Value: "TRUCK", Label: reference.RefLabel("drivers.power_plate", "TRUCK", lang)},
			{Value: "TRACTOR", Label: reference.RefLabel("drivers.power_plate", "TRACTOR", lang)},
		},
		TrailerPlateTypes: map[string][]ItemWithLabel{
			"TRUCK": {
				{Value: "FLATBED", Label: reference.RefLabel("drivers.trailer_truck", "FLATBED", lang)},
				{Value: "TENTED", Label: reference.RefLabel("drivers.trailer_truck", "TENTED", lang)},
				{Value: "BOX", Label: reference.RefLabel("drivers.trailer_truck", "BOX", lang)},
				{Value: "REEFER", Label: reference.RefLabel("drivers.trailer_truck", "REEFER", lang)},
				{Value: "TANKER", Label: reference.RefLabel("drivers.trailer_truck", "TANKER", lang)},
				{Value: "TIPPER", Label: reference.RefLabel("drivers.trailer_truck", "TIPPER", lang)},
				{Value: "CAR_CARRIER", Label: reference.RefLabel("drivers.trailer_truck", "CAR_CARRIER", lang)},
			},
			"TRACTOR": {
				{Value: "FLATBED", Label: reference.RefLabel("drivers.trailer_tractor", "FLATBED", lang)},
				{Value: "TENTED", Label: reference.RefLabel("drivers.trailer_tractor", "TENTED", lang)},
				{Value: "BOX", Label: reference.RefLabel("drivers.trailer_tractor", "BOX", lang)},
				{Value: "REEFER", Label: reference.RefLabel("drivers.trailer_tractor", "REEFER", lang)},
				{Value: "TANKER", Label: reference.RefLabel("drivers.trailer_tractor", "TANKER", lang)},
				{Value: "LOWBED", Label: reference.RefLabel("drivers.trailer_tractor", "LOWBED", lang)},
				{Value: "CONTAINER", Label: reference.RefLabel("drivers.trailer_tractor", "CONTAINER", lang)},
			},
		},
	}
	resp.OKLang(c, "ok", out)
}

// GetReferenceCargo возвращает справочник для раздела Cargo. value — верхний регистр, label и description — по X-Language.
func GetReferenceCargo(c *gin.Context) {
	lang := refLang(c)
	out := ReferenceCargoResponse{
		CargoStatus: []ItemWithLabelAndDescription{
			{Value: "CREATED", Label: reference.RefLabel("cargo.cargo_status", "CREATED", lang), Description: reference.CargoStatusDescription("CREATED", lang)},
			{Value: "PENDING_MODERATION", Label: reference.RefLabel("cargo.cargo_status", "PENDING_MODERATION", lang), Description: reference.CargoStatusDescription("PENDING_MODERATION", lang)},
			{Value: "SEARCHING", Label: reference.RefLabel("cargo.cargo_status", "SEARCHING", lang), Description: reference.CargoStatusDescription("SEARCHING", lang)},
			{Value: "REJECTED", Label: reference.RefLabel("cargo.cargo_status", "REJECTED", lang), Description: reference.CargoStatusDescription("REJECTED", lang)},
			{Value: "ASSIGNED", Label: reference.RefLabel("cargo.cargo_status", "ASSIGNED", lang), Description: reference.CargoStatusDescription("ASSIGNED", lang)},
			{Value: "IN_PROGRESS", Label: reference.RefLabel("cargo.cargo_status", "IN_PROGRESS", lang), Description: reference.CargoStatusDescription("IN_PROGRESS", lang)},
			{Value: "IN_TRANSIT", Label: reference.RefLabel("cargo.cargo_status", "IN_TRANSIT", lang), Description: reference.CargoStatusDescription("IN_TRANSIT", lang)},
			{Value: "DELIVERED", Label: reference.RefLabel("cargo.cargo_status", "DELIVERED", lang), Description: reference.CargoStatusDescription("DELIVERED", lang)},
			{Value: "COMPLETED", Label: reference.RefLabel("cargo.cargo_status", "COMPLETED", lang), Description: reference.CargoStatusDescription("COMPLETED", lang)},
			{Value: "CANCELLED", Label: reference.RefLabel("cargo.cargo_status", "CANCELLED", lang), Description: reference.CargoStatusDescription("CANCELLED", lang)},
		},
		RoutePointType: []ItemWithLabel{
			{Value: "LOAD", Label: reference.RefLabel("cargo.route_point_type", "LOAD", lang)},
			{Value: "UNLOAD", Label: reference.RefLabel("cargo.route_point_type", "UNLOAD", lang)},
			{Value: "CUSTOMS", Label: reference.RefLabel("cargo.route_point_type", "CUSTOMS", lang)},
			{Value: "TRANSIT", Label: reference.RefLabel("cargo.route_point_type", "TRANSIT", lang)},
		},
		OfferStatus: []ItemWithLabel{
			{Value: "PENDING", Label: reference.RefLabel("cargo.offer_status", "PENDING", lang)},
			{Value: "ACCEPTED", Label: reference.RefLabel("cargo.offer_status", "ACCEPTED", lang)},
			{Value: "REJECTED", Label: reference.RefLabel("cargo.offer_status", "REJECTED", lang)},
		},
		CreatedByType: []ItemWithLabel{
			{Value: "ADMIN", Label: reference.RefLabel("cargo.created_by_type", "ADMIN", lang)},
			{Value: "DISPATCHER", Label: reference.RefLabel("cargo.created_by_type", "DISPATCHER", lang)},
			{Value: "COMPANY", Label: reference.RefLabel("cargo.created_by_type", "COMPANY", lang)},
		},
		TruckType: []ItemWithLabel{
			{Value: "REFRIGERATOR", Label: reference.RefLabel("cargo.truck_type", "REFRIGERATOR", lang)},
			{Value: "TENT", Label: reference.RefLabel("cargo.truck_type", "TENT", lang)},
			{Value: "FLATBED", Label: reference.RefLabel("cargo.truck_type", "FLATBED", lang)},
			{Value: "TANKER", Label: reference.RefLabel("cargo.truck_type", "TANKER", lang)},
			{Value: "OTHER", Label: reference.RefLabel("cargo.truck_type", "OTHER", lang)},
		},
		TripStatus: []ItemWithLabel{
			{Value: "PENDING_DRIVER", Label: reference.RefLabel("cargo.trip_status", "PENDING_DRIVER", lang)},
			{Value: "ASSIGNED", Label: reference.RefLabel("cargo.trip_status", "ASSIGNED", lang)},
			{Value: "LOADING", Label: reference.RefLabel("cargo.trip_status", "LOADING", lang)},
			{Value: "EN_ROUTE", Label: reference.RefLabel("cargo.trip_status", "EN_ROUTE", lang)},
			{Value: "UNLOADING", Label: reference.RefLabel("cargo.trip_status", "UNLOADING", lang)},
			{Value: "COMPLETED", Label: reference.RefLabel("cargo.trip_status", "COMPLETED", lang)},
			{Value: "CANCELLED", Label: reference.RefLabel("cargo.trip_status", "CANCELLED", lang)},
		},
		ShipmentType:   refItemsToItemWithLabelLocalized(reference.ShipmentTypeRefs, "cargo.shipment_type", lang),
		Currency:       refItemsToItemWithLabelLocalized(reference.CurrencyRefs, "cargo.currency", lang),
		PrepaymentType: refItemsToItemWithLabelLocalized(reference.PrepaymentTypeRefs, "cargo.prepayment_type", lang),
		RemainingType:  refItemsToItemWithLabelLocalized(reference.RemainingTypeRefs, "cargo.remaining_type", lang),
		LoadingType:    refItemsToItemWithLabelLocalized(reference.LoadingTypeRefs, "cargo.loading_type", lang),
	}
	resp.OKLang(c, "ok", out)
}

// GetReferenceAdmin возвращает справочник для раздела Admin. value — верхний регистр, label — по X-Language.
func GetReferenceAdmin(c *gin.Context) {
	lang := refLang(c)
	out := ReferenceAdminResponse{
		AdminStatus: []ItemWithLabel{
			{Value: "ACTIVE", Label: reference.RefLabel("admin.admin_status", "ACTIVE", lang)},
			{Value: "INACTIVE", Label: reference.RefLabel("admin.admin_status", "INACTIVE", lang)},
			{Value: "BLOCKED", Label: reference.RefLabel("admin.admin_status", "BLOCKED", lang)},
		},
		AdminType: []ItemWithLabel{
			{Value: "CREATOR", Label: reference.RefLabel("admin.admin_type", "CREATOR", lang)},
			{Value: "OPERATOR", Label: reference.RefLabel("admin.admin_type", "OPERATOR", lang)},
		},
	}
	resp.OKLang(c, "ok", out)
}

// GetReferenceDispatchers возвращает справочник для раздела Freelance Dispatchers. value — верхний регистр, label — по X-Language.
func GetReferenceDispatchers(c *gin.Context) {
	lang := refLang(c)
	out := ReferenceDispatchersResponse{
		WorkStatus: []ItemWithLabel{
			{Value: "AVAILABLE", Label: reference.RefLabel("dispatchers.work_status", "AVAILABLE", lang)},
			{Value: "BUSY", Label: reference.RefLabel("dispatchers.work_status", "BUSY", lang)},
			{Value: "OFFLINE", Label: reference.RefLabel("dispatchers.work_status", "OFFLINE", lang)},
		},
	}
	resp.OKLang(c, "ok", out)
}

// GetReferenceCompany возвращает справочник для раздела Company (роли из БД). value — верхний регистр, label — по X-Language.
func GetReferenceCompany(rolesRepo *approles.Repo) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := refLang(c)
		roles, err := rolesRepo.ListAll(c.Request.Context())
		if err != nil {
			resp.ErrorLang(c, 500, "failed_to_load_roles")
			return
		}
		out := ReferenceCompanyResponse{
			CompanyType: []ItemWithLabel{
				{Value: "SHIPPER", Label: reference.RefLabel("company.company_type", "SHIPPER", lang)},
				{Value: "CARRIER", Label: reference.RefLabel("company.company_type", "CARRIER", lang)},
				{Value: "BROKER", Label: reference.RefLabel("company.company_type", "BROKER", lang)},
			},
			CompanyStatus: []ItemWithLabel{
				{Value: "ACTIVE", Label: reference.RefLabel("company.company_status", "ACTIVE", lang)},
				{Value: "INACTIVE", Label: reference.RefLabel("company.company_status", "INACTIVE", lang)},
				{Value: "BLOCKED", Label: reference.RefLabel("company.company_status", "BLOCKED", lang)},
				{Value: "PENDING", Label: reference.RefLabel("company.company_status", "PENDING", lang)},
			},
			CompanyUserRoles: []ItemWithLabel{
				{Value: "OWNER", Label: reference.RefLabel("company.role", "OWNER", lang)},
				{Value: "CEO", Label: reference.RefLabel("company.role", "CEO", lang)},
				{Value: "TOP_MANAGER", Label: reference.RefLabel("company.role", "TOP_MANAGER", lang)},
				{Value: "TOP_DISPATCHER", Label: reference.RefLabel("company.role", "TOP_DISPATCHER", lang)},
				{Value: "DISPATCHER", Label: reference.RefLabel("company.role", "DISPATCHER", lang)},
				{Value: "MANAGER", Label: reference.RefLabel("company.role", "MANAGER", lang)},
			},
			Roles: nil,
		}
		out.Roles = make([]RoleRef, 0, len(roles))
		for _, r := range roles {
			desc := ""
			if r.Description != nil {
				desc = *r.Description
			}
			nameUpper := strings.ToUpper(r.Name)
			label := reference.RefLabel("company.role", nameUpper, lang)
			if label == nameUpper {
				label = r.Name
			}
			out.Roles = append(out.Roles, RoleRef{ID: r.ID, Name: nameUpper, Label: label, Description: desc})
		}
		resp.OKLang(c, "ok", out)
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

type CityItem struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	CountryCode string   `json:"country_code"`
	Lat         *float64 `json:"lat,omitempty"`
	Lng         *float64 `json:"lng,omitempty"`
}

func cityNameByLang(c CityRef, lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	nameRu := strings.TrimSpace(c.NameRu)
	nameEn := ""
	if c.NameEn != nil {
		nameEn = strings.TrimSpace(*c.NameEn)
	}

	// Dataset is effectively ru/en.
	// Rule: if requested language is ru -> ru (fallback to en if ru empty).
	// Otherwise -> en (fallback to ru if en empty). This way uz/tr/zh won't unexpectedly return Russian.
	if lang == "ru" {
		if nameRu != "" {
			return nameRu
		}
		if nameEn != "" {
			return nameEn
		}
		return ""
	}

	if nameEn != "" {
		return nameEn
	}
	return nameRu
}

// GetReferenceCities возвращает справочник городов мира из встроенного датасета (in-memory, быстрый API).
// Query: country_code — фильтр по стране (UZ, AE, RU и т.д.). Данные: ~150k городов (lutangar/cities.json).
func GetReferenceCities() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Language")))
		switch lang {
		case "ru", "uz", "en", "tr", "zh":
		default:
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_x_language")
			return
		}

		countryCode := strings.TrimSpace(c.Query("country_code"))
		qRaw := strings.TrimSpace(c.Query("q"))
		qCode := strings.ToUpper(qRaw)
		qN := normRefSearch(qRaw)

		list, err := reference.CitiesByCountry(countryCode)
		if err != nil {
			resp.ErrorLang(c, 500, "failed_to_load_cities")
			return
		}

		type cityItemWithNames struct {
			item   CityItem
			nameRu string
			nameEn string
		}
		tmp := make([]cityItemWithNames, 0, len(list))
		for i := range list {
			ci := CityRef{
				ID:          list[i].ID,
				Code:        list[i].Code,
				NameRu:      list[i].NameRu,
				NameEn:      list[i].NameEn,
				CountryCode: list[i].CountryCode,
				Lat:         list[i].Lat,
				Lng:         list[i].Lng,
			}

			nameOut := cityNameByLang(ci, lang)
			nameRu := strings.TrimSpace(ci.NameRu)
			nameEn := ""
			if ci.NameEn != nil {
				nameEn = strings.TrimSpace(*ci.NameEn)
			}

			if qN != "" {
				codeMatch := strings.Contains(ci.Code, qCode) || strings.Contains(normRefSearch(ci.Code), qN)
				ruMatch := strings.Contains(normRefSearch(ci.NameRu), qN)
				enMatch := false
				if ci.NameEn != nil {
					enMatch = strings.Contains(normRefSearch(*ci.NameEn), qN)
				}
				if !codeMatch && !ruMatch && !enMatch {
					continue
				}
			}

			tmp = append(tmp, cityItemWithNames{
				item: CityItem{
					ID:          ci.ID,
					Code:        ci.Code,
					Name:        nameOut,
					CountryCode: ci.CountryCode,
					Lat:         ci.Lat,
					Lng:         ci.Lng,
				},
				nameRu: nameRu,
				nameEn: nameEn,
			})
		}

		if qN != "" {
			rank := func(it cityItemWithNames) int {
				codeU := it.item.Code
				codeN := normRefSearch(codeU)
				nameOutN := normRefSearch(it.item.Name)
				nameRuN := normRefSearch(it.nameRu)
				nameEnN := normRefSearch(it.nameEn)

				best := 9
				consider := func(field string, base int) {
					if field == "" || qN == "" {
						return
					}
					switch {
					case field == qN:
						if base < best {
							best = base
						}
					case strings.HasPrefix(field, qN):
						if base+2 < best {
							best = base + 2
						}
					case strings.Contains(field, qN):
						if base+4 < best {
							best = base + 4
						}
					}
				}

				if codeU == qCode {
					return 0
				}
				consider(codeN, 0)
				consider(nameOutN, 1)
				consider(nameRuN, 1)
				consider(nameEnN, 1)
				if best == 9 {
					return 9
				}
				return best
			}

			sort.SliceStable(tmp, func(i, j int) bool {
				ri := rank(tmp[i])
				rj := rank(tmp[j])
				if ri != rj {
					return ri < rj
				}
				if tmp[i].item.CountryCode != tmp[j].item.CountryCode {
					return tmp[i].item.CountryCode < tmp[j].item.CountryCode
				}
				if tmp[i].item.Code != tmp[j].item.Code {
					return tmp[i].item.Code < tmp[j].item.Code
				}
				return tmp[i].item.Name < tmp[j].item.Name
			})
		}

		items := make([]CityItem, 0, len(tmp))
		for _, t := range tmp {
			items = append(items, t.item)
		}
		resp.OKLang(c, "ok", gin.H{"items": items})
	}
}

type CountryItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func normRefSearch(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	// Make search more forgiving across scripts/punctuation.
	r := strings.NewReplacer(
		" ", "", "-", "", "_", "",
		"'", "", "’", "", "`", "",
		"ʻ", "", "ʼ", "",
		".", "", ",", "", "(", "", ")", "",
	)
	return r.Replace(s)
}

// GetReferenceCountries returns reference list of all country codes with localized name from X-Language header.
func GetReferenceCountries() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Language")))
		switch lang {
		case "ru", "uz", "en", "tr", "zh":
		default:
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_x_language")
			return
		}

		qRaw := strings.TrimSpace(c.Query("q"))
		q := strings.ToLower(qRaw)
		qCode := strings.ToUpper(qRaw)
		qN := normRefSearch(qRaw)

		all := reference.CountriesAll()
		items := make([]CountryItem, 0, len(all))
		for _, cc := range all {
			name := reference.CountryName(cc.Code, lang) // return name in selected language

			if q != "" {
				// Universal search: match against code + all supported languages, regardless of current X-Language.
				codeMatch := strings.Contains(cc.Code, qCode) || strings.Contains(normRefSearch(cc.Code), qN)

				nameRu := reference.CountryName(cc.Code, "ru")
				nameUz := reference.CountryName(cc.Code, "uz")
				nameEn := reference.CountryName(cc.Code, "en")
				nameTr := reference.CountryName(cc.Code, "tr")
				nameZh := reference.CountryName(cc.Code, "zh")

				nameMatch :=
					strings.Contains(normRefSearch(nameRu), qN) ||
						strings.Contains(normRefSearch(nameUz), qN) ||
						strings.Contains(normRefSearch(nameEn), qN) ||
						strings.Contains(normRefSearch(nameTr), qN) ||
						strings.Contains(normRefSearch(nameZh), qN)

				if !codeMatch && !nameMatch {
					continue
				}
			}
			items = append(items, CountryItem{
				Code: cc.Code,
				Name: name,
			})
		}

		if q != "" {
			rank := func(it CountryItem) int {
				codeU := it.Code
				codeN := normRefSearch(codeU)

				// Use all languages for ranking too (universal search), but return Name in selected language.
				nameRuN := normRefSearch(reference.CountryName(codeU, "ru"))
				nameUzN := normRefSearch(reference.CountryName(codeU, "uz"))
				nameEnN := normRefSearch(reference.CountryName(codeU, "en"))
				nameTrN := normRefSearch(reference.CountryName(codeU, "tr"))
				nameZhN := normRefSearch(reference.CountryName(codeU, "zh"))

				best := 9
				consider := func(field string, exactRank int) {
					if field == "" || qN == "" {
						return
					}
					switch {
					case field == qN:
						if exactRank < best {
							best = exactRank
						}
					case strings.HasPrefix(field, qN):
						if exactRank+2 < best {
							best = exactRank + 2
						}
					case strings.Contains(field, qN):
						if exactRank+4 < best {
							best = exactRank + 4
						}
					}
				}

				// code is always the strongest signal
				if codeU == qCode {
					return 0
				}
				consider(codeN, 0)
				consider(nameRuN, 1)
				consider(nameUzN, 1)
				consider(nameEnN, 1)
				consider(nameTrN, 1)
				consider(nameZhN, 1)

				if best == 9 {
					return 9
				}
				return best
			}
			sort.SliceStable(items, func(i, j int) bool {
				ri := rank(items[i])
				rj := rank(items[j])
				if ri != rj {
					return ri < rj
				}
				if items[i].Code != items[j].Code {
					return items[i].Code < items[j].Code
				}
				return items[i].Name < items[j].Name
			})
		}

		resp.OKLang(c, "ok", gin.H{"items": items})
	}
}
