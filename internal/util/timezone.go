package util

import (
	"sync"
	"time"

	_ "time/tzdata"
)

var (
	tashkentOnce sync.Once
	tashkentLoc  *time.Location
)

func tashkentLocation() *time.Location {
	tashkentOnce.Do(func() {
		loc, err := time.LoadLocation("Asia/Tashkent")
		if err != nil {
			// Uzbekistan does not use DST; fixed offset is a safe fallback.
			tashkentLoc = time.FixedZone("Asia/Tashkent", 5*60*60)
			return
		}
		tashkentLoc = loc
	})
	return tashkentLoc
}

// InTashkent converts time to Asia/Tashkent location (UTC+5).
func InTashkent(t time.Time) time.Time {
	loc := tashkentLocation()
	if loc == nil {
		return t
	}
	return t.In(loc)
}

