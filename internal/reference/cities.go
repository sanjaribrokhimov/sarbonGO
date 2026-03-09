// Package reference provides in-memory reference data (world cities) for fast API.
// Uses github.com/tidwall/cities (~10k cities) + supplemental list for UZ, AE, TM, KG, TJ.
package reference

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/tidwall/cities"
)

var (
	citiesOnce   sync.Once
	citiesList   []CityRef
)

// CityRef — элемент справочника городов для API.
type CityRef struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	NameRu      string   `json:"name_ru"`
	NameEn      *string  `json:"name_en,omitempty"`
	CountryCode string   `json:"country_code"`
	Lat         *float64 `json:"lat,omitempty"`
	Lng         *float64 `json:"lng,omitempty"`
}

var alphaOnly = regexp.MustCompile(`[a-zA-Z]+`)

func codeFromName(name string) string {
	parts := alphaOnly.FindAllString(name, -1)
	var b strings.Builder
	for _, part := range parts {
		for _, r := range part {
			if unicode.IsLetter(r) {
				b.WriteRune(unicode.ToUpper(r))
				if b.Len() >= 3 {
					return b.String()
				}
			}
		}
	}
	return b.String()
}

// LoadCities builds the full cities list from tidwall/cities + supplemental once.
func LoadCities() ([]CityRef, error) {
	citiesOnce.Do(func() {
		seen := make(map[string]map[string]int) // countryCode -> code -> count (for dup suffix)
		var list []CityRef

		// 1) From tidwall/cities
		for i := range cities.Cities {
			c := &cities.Cities[i]
			countryCode := countryNameToAlpha2[c.Country]
			if countryCode == "" {
				countryCode = "XX"
			}
			code := codeFromName(c.City)
			if code == "" {
				code = "XXX"
			}
			if seen[countryCode] == nil {
				seen[countryCode] = make(map[string]int)
			}
			n := seen[countryCode][code]
			seen[countryCode][code] = n + 1
			if n > 0 {
				code = code + fmt.Sprintf("%d", n)
				if len(code) > 8 {
					code = code[:8]
				}
			}
			lat, lng := c.Latitude, c.Longitude
			list = append(list, CityRef{
				ID:          code + "-" + countryCode,
				Code:        code,
				NameRu:      c.City,
				NameEn:      &c.City,
				CountryCode: countryCode,
				Lat:         &lat,
				Lng:         &lng,
			})
		}

		// 2) Supplemental (UZ, AE, TM, KG, TJ)
		list = append(list, supplementalCities...)
		citiesList = list
	})
	return citiesList, nil
}

// CitiesByCountry returns cities filtered by country code (empty = all).
func CitiesByCountry(countryCode string) ([]CityRef, error) {
	list, err := LoadCities()
	if err != nil {
		return nil, err
	}
	if countryCode == "" {
		return list, nil
	}
	out := make([]CityRef, 0, 256)
	for _, c := range list {
		if c.CountryCode == countryCode {
			out = append(out, c)
		}
	}
	return out, nil
}
