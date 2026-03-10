// Package timeutil предоставляет время по Ташкенту (Asia/Tashkent) для расчётов в приложении.
package timeutil

import (
	"sync"
	"time"
)

var (
	tashkentLocation *time.Location
	tzOnce           sync.Once
	tzErr            error
)

// Tashkent возвращает *time.Location для Asia/Tashkent. При ошибке загрузки возвращает time.UTC.
func Tashkent() *time.Location {
	tzOnce.Do(func() {
		tashkentLocation, tzErr = time.LoadLocation("Asia/Tashkent")
		if tzErr != nil {
			tashkentLocation = time.UTC
		}
	})
	return tashkentLocation
}

// NowTashkent возвращает текущий момент времени с локацией Ташкент (для расчётов expires_at и т.д.).
// Момент в времени (instant) тот же, что time.Now(); локация влияет только на отображение (часы, дата).
func NowTashkent() time.Time {
	return time.Now().In(Tashkent())
}
