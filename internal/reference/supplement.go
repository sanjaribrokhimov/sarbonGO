package reference

// supplementalCities — города для стран, которых нет в tidwall/cities (Узбекистан, ОАЭ, и др.).
var supplementalCities = []CityRef{
	// Uzbekistan (UZ)
	{ID: "TAS-UZ", Code: "TAS", NameRu: "Ташкент", NameEn: strPtr("Tashkent"), CountryCode: "UZ", Lat: f64(41.311081), Lng: f64(69.240562)},
	{ID: "SAM-UZ", Code: "SAM", NameRu: "Самарканд", NameEn: strPtr("Samarkand"), CountryCode: "UZ", Lat: f64(39.654167), Lng: f64(66.959722)},
	{ID: "AND-UZ", Code: "AND", NameRu: "Андижан", NameEn: strPtr("Andijan"), CountryCode: "UZ", Lat: f64(40.783333), Lng: f64(72.333333)},
	{ID: "BUK-UZ", Code: "BUK", NameRu: "Бухара", NameEn: strPtr("Bukhara"), CountryCode: "UZ", Lat: f64(39.774722), Lng: f64(64.428611)},
	{ID: "FER-UZ", Code: "FER", NameRu: "Фергана", NameEn: strPtr("Fergana"), CountryCode: "UZ", Lat: f64(40.386389), Lng: f64(71.786389)},
	{ID: "NAM-UZ", Code: "NAM", NameRu: "Наманган", NameEn: strPtr("Namangan"), CountryCode: "UZ", Lat: f64(40.998333), Lng: f64(71.672778)},
	{ID: "NUK-UZ", Code: "NUK", NameRu: "Нукус", NameEn: strPtr("Nukus"), CountryCode: "UZ", Lat: f64(42.453056), Lng: f64(59.610278)},
	{ID: "QAR-UZ", Code: "QAR", NameRu: "Карши", NameEn: strPtr("Qarshi"), CountryCode: "UZ", Lat: f64(38.861111), Lng: f64(65.789167)},
	{ID: "TER-UZ", Code: "TER", NameRu: "Термез", NameEn: strPtr("Termez"), CountryCode: "UZ", Lat: f64(37.224167), Lng: f64(67.278333)},
	// United Arab Emirates (AE)
	{ID: "DXB-AE", Code: "DXB", NameRu: "Дубай", NameEn: strPtr("Dubai"), CountryCode: "AE", Lat: f64(25.204849), Lng: f64(55.270783)},
	{ID: "AUH-AE", Code: "AUH", NameRu: "Абу-Даби", NameEn: strPtr("Abu Dhabi"), CountryCode: "AE", Lat: f64(24.453889), Lng: f64(54.377343)},
	{ID: "SHJ-AE", Code: "SHJ", NameRu: "Шарджа", NameEn: strPtr("Sharjah"), CountryCode: "AE", Lat: f64(25.357308), Lng: f64(55.403304)},
	{ID: "AJM-AE", Code: "AJM", NameRu: "Аджман", NameEn: strPtr("Ajman"), CountryCode: "AE", Lat: f64(25.41111), Lng: f64(55.43504)},
	{ID: "RAK-AE", Code: "RAK", NameRu: "Рас-эль-Хайма", NameEn: strPtr("Ras Al Khaimah"), CountryCode: "AE", Lat: f64(25.78946), Lng: f64(55.9432)},
	{ID: "FJR-AE", Code: "FJR", NameRu: "Фуджейра", NameEn: strPtr("Fujairah"), CountryCode: "AE", Lat: f64(25.12841), Lng: f64(56.32646)},
	// Turkmenistan (TM)
	{ID: "ASB-TM", Code: "ASB", NameRu: "Ашхабад", NameEn: strPtr("Ashgabat"), CountryCode: "TM", Lat: f64(37.95), Lng: f64(58.383333)},
	{ID: "TUR-TM", Code: "TUR", NameRu: "Туркменабад", NameEn: strPtr("Turkmenabat"), CountryCode: "TM", Lat: f64(39.07328), Lng: f64(63.57861)},
	// Kyrgyzstan (KG)
	{ID: "FRU-KG", Code: "FRU", NameRu: "Бишкек", NameEn: strPtr("Bishkek"), CountryCode: "KG", Lat: f64(42.874722), Lng: f64(74.612222)},
	{ID: "OSS-KG", Code: "OSS", NameRu: "Ош", NameEn: strPtr("Osh"), CountryCode: "KG", Lat: f64(40.52828), Lng: f64(72.7985)},
	// Tajikistan (TJ)
	{ID: "DUS-TJ", Code: "DUS", NameRu: "Душанбе", NameEn: strPtr("Dushanbe"), CountryCode: "TJ", Lat: f64(38.559772), Lng: f64(68.773928)},
	{ID: "KHO-TJ", Code: "KHO", NameRu: "Худжанд", NameEn: strPtr("Khujand"), CountryCode: "TJ", Lat: f64(40.28256), Lng: f64(69.62217)},
}

func strPtr(s string) *string { return &s }
func f64(f float64) *float64 { return &f }
