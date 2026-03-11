// Package reference: справочники для грузов (truck_type, route_point type, shipment_type, валюты, оплата, способы погрузки).
// Используются в GET /v1/reference/cargo и для валидации при создании/обновлении груза.

package reference

import "strings"

// RefItem — value (код для API) и label (подпись для UI). В API принимается value в любом регистре.
type RefItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// RoutePointTypeRefs — тип точки маршрута (load, unload, customs, transit).
var RoutePointTypeRefs = []RefItem{
	{Value: "load", Label: "Погрузка"},
	{Value: "unload", Label: "Выгрузка"},
	{Value: "customs", Label: "Таможня"},
	{Value: "transit", Label: "Транзит"},
}

// TruckTypeRefs — тип кузова (значения как в API — нижний регистр).
var TruckTypeRefs = []RefItem{
	{Value: "refrigerator", Label: "Рефрижератор"},
	{Value: "tent", Label: "Тент"},
	{Value: "flatbed", Label: "Борт"},
	{Value: "tanker", Label: "Цистерна"},
	{Value: "other", Label: "Другое"},
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

// PrepaymentTypeRefs — способ предоплаты.
var PrepaymentTypeRefs = []RefItem{
	{Value: "bank_transfer", Label: "Банковский перевод"},
	{Value: "cash", Label: "Наличные"},
	{Value: "card", Label: "Карта"},
	{Value: "invoice", Label: "По счёту"},
	{Value: "other", Label: "Другое"},
}

// RemainingTypeRefs — условия оплаты остатка.
var RemainingTypeRefs = []RefItem{
	{Value: "on_delivery", Label: "По факту выгрузки"},
	{Value: "after_invoice", Label: "По счёту после выгрузки"},
	{Value: "cash", Label: "Наличными при выгрузке"},
	{Value: "deferred", Label: "Отсрочка платежа"},
	{Value: "other", Label: "Другое"},
}

// LoadingTypeRefs — способы погрузки (loading_types).
var LoadingTypeRefs = []RefItem{
	{Value: "top", Label: "Верхняя"},
	{Value: "side", Label: "Боковая"},
	{Value: "rear", Label: "Задняя"},
	{Value: "crane", Label: "Кран"},
	{Value: "forklift", Label: "Погрузчик"},
	{Value: "other", Label: "Другое"},
}

// AllowedValues возвращает слайс допустимых value в нижнем регистре (для валидации).
func AllowedValues(items []RefItem) []string {
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, strings.ToLower(i.Value))
	}
	return out
}

// AllowedShipmentTypes возвращает допустимые shipment_type (нижний регистр).
func AllowedShipmentTypes() []string { return AllowedValues(ShipmentTypeRefs) }

// AllowedCurrencies возвращает допустимые валюты (нижний регистр).
func AllowedCurrencies() []string { return AllowedValues(CurrencyRefs) }

// AllowedPrepaymentTypes возвращает допустимые prepayment_type (нижний регистр).
func AllowedPrepaymentTypes() []string { return AllowedValues(PrepaymentTypeRefs) }

// AllowedRemainingTypes возвращает допустимые remaining_type (нижний регистр).
func AllowedRemainingTypes() []string { return AllowedValues(RemainingTypeRefs) }

// AllowedLoadingTypes возвращает допустимые loading_types (нижний регистр).
func AllowedLoadingTypes() []string { return AllowedValues(LoadingTypeRefs) }

// AllowedRoutePointTypes возвращает допустимые type точки маршрута (load, unload, customs, transit).
func AllowedRoutePointTypes() []string { return AllowedValues(RoutePointTypeRefs) }

// AllowedTruckTypes возвращает допустимые truck_type (нижний регистр).
func AllowedTruckTypes() []string { return AllowedValues(TruckTypeRefs) }

// IsAllowed проверяет, что value есть в списке (сравнение без учёта регистра).
func IsAllowed(value string, allowed []string) bool {
	v := strings.TrimSpace(strings.ToLower(value))
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
