// Package reference: справочники для грузов (truck_type, route_point type, shipment_type, валюты, оплата, способы погрузки).
// Используются в GET /v1/reference/cargo и для валидации при создании/обновлении груза.

package reference

import "strings"

// RefItem — value (код для API) и label (подпись для UI). В API принимается value в любом регистре.
type RefItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// RoutePointTypeRefs — тип точки маршрута (UPPERCASE в API и справочнике).
var RoutePointTypeRefs = []RefItem{
	{Value: "LOAD", Label: "Погрузка"},
	{Value: "UNLOAD", Label: "Выгрузка"},
	{Value: "CUSTOMS", Label: "Таможня"},
	{Value: "TRANSIT", Label: "Транзит"},
}

// TruckTypeRefs — тип кузова (UPPERCASE в API и справочнике).
var TruckTypeRefs = []RefItem{
	{Value: "REFRIGERATOR", Label: "Рефрижератор"},
	{Value: "TENT", Label: "Тент"},
	{Value: "FLATBED", Label: "Борт"},
	{Value: "TANKER", Label: "Цистерна"},
	{Value: "OTHER", Label: "Другое"},
}

// ShipmentTypeRefs — тип отправки (FTL, LTL и т.д.).
var ShipmentTypeRefs = []RefItem{
	{Value: "FTL", Label: "Полная загрузка (FTL)"},
	{Value: "LTL", Label: "Частичная загрузка (LTL)"},
	{Value: "PACKAGE", Label: "Посылка"},
	{Value: "OTHER", Label: "Другое"},
}

// CurrencyRefs — валюты (total_currency, prepayment_currency, remaining_currency).
var CurrencyRefs = []RefItem{
	{Value: "USD", Label: "USD (доллар США)"},
	{Value: "UZS", Label: "UZS (сум)"},
	{Value: "EUR", Label: "EUR (евро)"},
	{Value: "RUB", Label: "RUB (рубль)"},
	{Value: "KZT", Label: "KZT (тенге)"},
	{Value: "AED", Label: "AED (дирхам ОАЭ)"},
	{Value: "OTHER", Label: "Другая"},
}

// PrepaymentTypeRefs — способ предоплаты (UPPERCASE).
var PrepaymentTypeRefs = []RefItem{
	{Value: "BANK_TRANSFER", Label: "Банковский перевод"},
	{Value: "CASH", Label: "Наличные"},
	{Value: "CARD", Label: "Карта"},
	{Value: "INVOICE", Label: "По счёту"},
	{Value: "OTHER", Label: "Другое"},
}

// RemainingTypeRefs — условия оплаты остатка (UPPERCASE).
var RemainingTypeRefs = []RefItem{
	{Value: "ON_DELIVERY", Label: "По факту выгрузки"},
	{Value: "AFTER_INVOICE", Label: "По счёту после выгрузки"},
	{Value: "CASH", Label: "Наличными при выгрузке"},
	{Value: "DEFERRED", Label: "Отсрочка платежа"},
	{Value: "OTHER", Label: "Другое"},
}

// LoadingTypeRefs — способы погрузки (UPPERCASE).
var LoadingTypeRefs = []RefItem{
	{Value: "TOP", Label: "Верхняя"},
	{Value: "SIDE", Label: "Боковая"},
	{Value: "REAR", Label: "Задняя"},
	{Value: "CRANE", Label: "Кран"},
	{Value: "FORKLIFT", Label: "Погрузчик"},
	{Value: "OTHER", Label: "Другое"},
}

// PackagingTypeRefs — тип упаковки (UPPERCASE).
// Используется в UI как dropdown/hint для поля cargo.packaging.
var PackagingTypeRefs = []RefItem{
	{Value: "BAG", Label: "Пакет"},
	{Value: "PALLETS", Label: "Палеты"},
	{Value: "BULK", Label: "Россыпью"},
	{Value: "BOXES", Label: "Коробки"},
	{Value: "HEAP", Label: "Навалом"},
}

// AllowedValues возвращает слайс допустимых value в ВЕРХНЕМ регистре (для валидации и хранения).
func AllowedValues(items []RefItem) []string {
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, strings.ToUpper(strings.TrimSpace(i.Value)))
	}
	return out
}

// AllowedShipmentTypes возвращает допустимые shipment_type (UPPERCASE).
func AllowedShipmentTypes() []string { return AllowedValues(ShipmentTypeRefs) }

// AllowedCurrencies возвращает допустимые валюты (UPPERCASE).
func AllowedCurrencies() []string { return AllowedValues(CurrencyRefs) }

// AllowedPrepaymentTypes возвращает допустимые prepayment_type (UPPERCASE).
func AllowedPrepaymentTypes() []string { return AllowedValues(PrepaymentTypeRefs) }

// AllowedRemainingTypes возвращает допустимые remaining_type (UPPERCASE).
func AllowedRemainingTypes() []string { return AllowedValues(RemainingTypeRefs) }

// AllowedLoadingTypes возвращает допустимые loading_types (UPPERCASE).
func AllowedLoadingTypes() []string { return AllowedValues(LoadingTypeRefs) }

// AllowedPackagingTypes возвращает допустимые packaging_type (UPPERCASE).
func AllowedPackagingTypes() []string { return AllowedValues(PackagingTypeRefs) }

// AllowedRoutePointTypes возвращает допустимые type точки маршрута (UPPERCASE).
func AllowedRoutePointTypes() []string { return AllowedValues(RoutePointTypeRefs) }

// AllowedTruckTypes возвращает допустимые truck_type (UPPERCASE).
func AllowedTruckTypes() []string { return AllowedValues(TruckTypeRefs) }

// IsAllowed проверяет, что value есть в списке (приводит к верхнему регистру для сравнения).
func IsAllowed(value string, allowed []string) bool {
	v := strings.ToUpper(strings.TrimSpace(value))
	if v == "" {
		return false
	}
	for _, a := range allowed {
		if a == v {
			return true
		}
	}
	return false
}
