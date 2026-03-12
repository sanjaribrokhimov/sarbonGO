package reference

import (
	"sort"
	"strings"
)

// CountryRef is a reference item for country codes (ISO 3166-1 alpha-2).
// Names are optional; if a translation is missing we fallback to English.
type CountryRef struct {
	Code   string  `json:"code"`
	NameRu *string `json:"name_ru,omitempty"`
	NameUz *string `json:"name_uz,omitempty"`
	NameEn *string `json:"name_en,omitempty"`
	NameTr *string `json:"name_tr,omitempty"`
	NameZh *string `json:"name_zh,omitempty"`
}

// countriesByCode contains curated translations for the most used countries in Sarbon.
// For the rest we fall back to English names from the cities dataset mapping.
var countriesByCode = map[string]CountryRef{
	"UZ": {Code: "UZ", NameRu: strPtr("Узбекистан"), NameUz: strPtr("Oʻzbekiston"), NameEn: strPtr("Uzbekistan"), NameTr: strPtr("Özbekistan"), NameZh: strPtr("乌兹别克斯坦")},
	"RU": {Code: "RU", NameRu: strPtr("Россия"), NameUz: strPtr("Rossiya"), NameEn: strPtr("Russia"), NameTr: strPtr("Rusya"), NameZh: strPtr("俄罗斯")},
	"KZ": {Code: "KZ", NameRu: strPtr("Казахстан"), NameUz: strPtr("Qozogʻiston"), NameEn: strPtr("Kazakhstan"), NameTr: strPtr("Kazakistan"), NameZh: strPtr("哈萨克斯坦")},
	"KG": {Code: "KG", NameRu: strPtr("Кыргызстан"), NameUz: strPtr("Qirgʻiziston"), NameEn: strPtr("Kyrgyzstan"), NameTr: strPtr("Kırgızistan"), NameZh: strPtr("吉尔吉斯斯坦")},
	"TJ": {Code: "TJ", NameRu: strPtr("Таджикистан"), NameUz: strPtr("Tojikiston"), NameEn: strPtr("Tajikistan"), NameTr: strPtr("Tacikistan"), NameZh: strPtr("塔吉克斯坦")},
	"TM": {Code: "TM", NameRu: strPtr("Туркменистан"), NameUz: strPtr("Turkmaniston"), NameEn: strPtr("Turkmenistan"), NameTr: strPtr("Türkmenistan"), NameZh: strPtr("土库曼斯坦")},
	"AE": {Code: "AE", NameRu: strPtr("ОАЭ"), NameUz: strPtr("BAA"), NameEn: strPtr("United Arab Emirates"), NameTr: strPtr("Birleşik Arap Emirlikleri"), NameZh: strPtr("阿联酋")},
	"TR": {Code: "TR", NameRu: strPtr("Турция"), NameUz: strPtr("Turkiya"), NameEn: strPtr("Turkey"), NameTr: strPtr("Türkiye"), NameZh: strPtr("土耳其")},
	"CN": {Code: "CN", NameRu: strPtr("Китай"), NameUz: strPtr("Xitoy"), NameEn: strPtr("China"), NameTr: strPtr("Çin"), NameZh: strPtr("中国")},
}

var countryCodeToEnglishName map[string]string

func init() {
	// Reverse map: alpha-2 code -> English name (unique enough for our dataset).
	// When multiple names map to same code, prefer the first encountered.
	c2n := make(map[string]string, len(countryNameToAlpha2))
	for name, code := range countryNameToAlpha2 {
		if _, ok := c2n[code]; ok {
			continue
		}
		c2n[code] = name
	}
	countryCodeToEnglishName = c2n
}

// CountriesAll returns a stable sorted list of all known country codes (from the cities dataset),
// enriched with curated translations where available.
func CountriesAll() []CountryRef {
	seen := make(map[string]struct{}, len(countryCodeToEnglishName))
	out := make([]CountryRef, 0, len(countryCodeToEnglishName))

	for code, en := range countryCodeToEnglishName {
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}

		if curated, ok := countriesByCode[code]; ok {
			out = append(out, curated)
			continue
		}

		en2 := en
		out = append(out, CountryRef{
			Code:   code,
			NameEn: &en2,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// CountryName returns a localized country name for the given code and language.
// lang must be one of ru, uz, en, tr, zh; fallback is English, then Russian, then code.
func CountryName(code string, lang string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	lang = strings.ToLower(strings.TrimSpace(lang))
	if code == "" {
		return ""
	}
	ref, ok := countriesByCode[code]
	if !ok {
		if en, ok2 := countryCodeToEnglishName[code]; ok2 && strings.TrimSpace(en) != "" {
			return en
		}
		return code
	}

	pick := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.TrimSpace(*p)
	}

	switch lang {
	case "ru":
		if v := pick(ref.NameRu); v != "" {
			return v
		}
	case "uz":
		if v := pick(ref.NameUz); v != "" {
			return v
		}
	case "en":
		if v := pick(ref.NameEn); v != "" {
			return v
		}
	case "tr":
		if v := pick(ref.NameTr); v != "" {
			return v
		}
	case "zh":
		if v := pick(ref.NameZh); v != "" {
			return v
		}
	}

	if v := pick(ref.NameEn); v != "" {
		return v
	}
	if v := pick(ref.NameRu); v != "" {
		return v
	}
	return code
}

