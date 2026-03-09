package reference

import (
	"testing"
)

func TestLoadCities(t *testing.T) {
	list, err := LoadCities()
	if err != nil {
		t.Fatalf("LoadCities: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("LoadCities: empty list")
	}
	// Supplemental: UZ, AE must be present
	var hasUZ, hasAE bool
	var tashkent, dubai CityRef
	for _, c := range list {
		if c.CountryCode == "UZ" {
			hasUZ = true
			if c.Code == "TAS" {
				tashkent = c
			}
		}
		if c.CountryCode == "AE" {
			hasAE = true
			if c.Code == "DXB" {
				dubai = c
			}
		}
	}
	if !hasUZ {
		t.Error("expected Uzbekistan cities (UZ)")
	}
	if !hasAE {
		t.Error("expected UAE cities (AE)")
	}
	if tashkent.Code != "TAS" || tashkent.NameRu != "Ташкент" {
		t.Errorf("Tashkent: got code=%q name_ru=%q", tashkent.Code, tashkent.NameRu)
	}
	if dubai.Code != "DXB" || dubai.NameEn == nil || *dubai.NameEn != "Dubai" {
		t.Errorf("Dubai: got code=%q name_en=%v", dubai.Code, dubai.NameEn)
	}
}

func TestCitiesByCountry(t *testing.T) {
	uz, err := CitiesByCountry("UZ")
	if err != nil {
		t.Fatalf("CitiesByCountry(UZ): %v", err)
	}
	if len(uz) < 9 {
		t.Errorf("expected at least 9 cities for UZ, got %d", len(uz))
	}
	ae, err := CitiesByCountry("AE")
	if err != nil {
		t.Fatalf("CitiesByCountry(AE): %v", err)
	}
	if len(ae) < 6 {
		t.Errorf("expected at least 6 cities for AE, got %d", len(ae))
	}
	all, err := CitiesByCountry("")
	if err != nil {
		t.Fatalf("CitiesByCountry(empty): %v", err)
	}
	if len(all) < 10000 {
		t.Errorf("expected 10000+ cities total, got %d", len(all))
	}
}
