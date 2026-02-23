package handlers

import (
	"github.com/gin-gonic/gin"

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

// TransportOptionsResponse — ответ GET /v1/transport-options.
type TransportOptionsResponse struct {
	PowerPlateTypes        []PowerPlateOption            `json:"power_plate_types"`
	TrailerPlateTypesByPower map[string][]TrailerPlateOption `json:"trailer_plate_types_by_power"`
}

var (
	powerPlateTypes = []PowerPlateOption{
		{Code: "TRUCK", Label: "Грузовик + прицеп"},
		{Code: "TRACTOR", Label: "Тягач + полуприцеп"},
	}

	trailerForTruck = []TrailerPlateOption{
		{Code: "FLATBED", Label: "Бортовой прицеп"},
		{Code: "TENTED", Label: "Тентованный прицеп"},
		{Code: "BOX", Label: "Фургонный прицеп"},
		{Code: "REEFER", Label: "Рефрижераторный прицеп"},
		{Code: "TANKER", Label: "Прицеп-цистерна"},
		{Code: "TIPPER", Label: "Самосвальный прицеп"},
		{Code: "CAR_CARRIER", Label: "Прицеп-автовоз"},
	}

	trailerForTractor = []TrailerPlateOption{
		{Code: "FLATBED", Label: "Бортовой полуприцеп"},
		{Code: "TENTED", Label: "Тентованный полуприцеп"},
		{Code: "BOX", Label: "Фургонный полуприцеп"},
		{Code: "REEFER", Label: "Рефрижераторный полуприцеп"},
		{Code: "TANKER", Label: "Полуприцеп-цистерна"},
		{Code: "LOWBED", Label: "Низкорамный полуприцеп"},
		{Code: "CONTAINER", Label: "Полуприцеп-контейнеровоз"},
	}
)

// GetTransportOptions возвращает списки power_plate_type и trailer_plate_type для UI.
// trailer_plate_type зависит от выбранного power_plate_type:
// - TRUCK → прицепы (FLATBED, TENTED, BOX, REEFER, TANKER, TIPPER, CAR_CARRIER)
// - TRACTOR → полуприцепы (FLATBED, TENTED, BOX, REEFER, TANKER, LOWBED, CONTAINER)
func GetTransportOptions(c *gin.Context) {
	resp.OK(c, TransportOptionsResponse{
		PowerPlateTypes: powerPlateTypes,
		TrailerPlateTypesByPower: map[string][]TrailerPlateOption{
			"TRUCK":   trailerForTruck,
			"TRACTOR": trailerForTractor,
		},
	})
}
