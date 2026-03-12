package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"sarbonNew/internal/reference"
	"sarbonNew/internal/server/resp"
)

// PowerPlateOption — тип основной машины (Base Vehicle).
type PowerPlateOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// TrailerPlateOption — тип прицепной части (зависит от power_plate_type).
type TrailerPlateOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// TransportOptionsResponse — ответ GET /v1/driver/transport-options.
type TransportOptionsResponse struct {
	PowerPlateTypes        []PowerPlateOption            `json:"power_plate_types"`
	TrailerPlateTypesByPower map[string][]TrailerPlateOption `json:"trailer_plate_types_by_power"`
}

// refLangTransport reads X-Language (ru, uz, en, tr, zh). Returns "en" if invalid or missing.
func refLangTransport(c *gin.Context) string {
	lang := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Language")))
	switch lang {
	case "ru", "uz", "en", "tr", "zh":
		return lang
	}
	return "en"
}

// GetTransportOptions возвращает списки power_plate_type и trailer_plate_type для UI.
// Labels по заголовку X-Language. trailer_plate_type зависит от выбранного power_plate_type:
// - TRUCK → прицепы (FLATBED, TENTED, BOX, REEFER, TANKER, TIPPER, CAR_CARRIER)
// - TRACTOR → полуприцепы (FLATBED, TENTED, BOX, REEFER, TANKER, LOWBED, CONTAINER)
func GetTransportOptions(c *gin.Context) {
	lang := refLangTransport(c)
	resp.OK(c, TransportOptionsResponse{
		PowerPlateTypes: []PowerPlateOption{
			{Code: "TRUCK", Label: reference.RefLabel("transport.power_plate", "TRUCK", lang)},
			{Code: "TRACTOR", Label: reference.RefLabel("transport.power_plate", "TRACTOR", lang)},
		},
		TrailerPlateTypesByPower: map[string][]TrailerPlateOption{
			"TRUCK": {
				{Code: "FLATBED", Label: reference.RefLabel("transport.trailer_truck", "FLATBED", lang)},
				{Code: "TENTED", Label: reference.RefLabel("transport.trailer_truck", "TENTED", lang)},
				{Code: "BOX", Label: reference.RefLabel("transport.trailer_truck", "BOX", lang)},
				{Code: "REEFER", Label: reference.RefLabel("transport.trailer_truck", "REEFER", lang)},
				{Code: "TANKER", Label: reference.RefLabel("transport.trailer_truck", "TANKER", lang)},
				{Code: "TIPPER", Label: reference.RefLabel("transport.trailer_truck", "TIPPER", lang)},
				{Code: "CAR_CARRIER", Label: reference.RefLabel("transport.trailer_truck", "CAR_CARRIER", lang)},
			},
			"TRACTOR": {
				{Code: "FLATBED", Label: reference.RefLabel("transport.trailer_tractor", "FLATBED", lang)},
				{Code: "TENTED", Label: reference.RefLabel("transport.trailer_tractor", "TENTED", lang)},
				{Code: "BOX", Label: reference.RefLabel("transport.trailer_tractor", "BOX", lang)},
				{Code: "REEFER", Label: reference.RefLabel("transport.trailer_tractor", "REEFER", lang)},
				{Code: "TANKER", Label: reference.RefLabel("transport.trailer_tractor", "TANKER", lang)},
				{Code: "LOWBED", Label: reference.RefLabel("transport.trailer_tractor", "LOWBED", lang)},
				{Code: "CONTAINER", Label: reference.RefLabel("transport.trailer_tractor", "CONTAINER", lang)},
			},
		},
	})
}
