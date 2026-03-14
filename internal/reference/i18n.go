// Package reference: translations for reference API (label + description).
// X-Language: ru, uz, en, tr, zh. Value in API is always uppercase.

package reference

import "strings"

// RefLabel returns label for "section.value". value is normalized to uppercase for lookup. Fallback: en -> ru -> value.
func RefLabel(section, value, lang string) string {
	key := section + "." + strings.ToUpper(strings.TrimSpace(value))
	if key == "." {
		return value
	}
	lang = strings.ToLower(strings.TrimSpace(lang))
	labels, ok := refLabels[key]
	if !ok {
		return value
	}
	if l, ok := labels[lang]; ok && l != "" {
		return l
	}
	if l, ok := labels["en"]; ok && l != "" {
		return l
	}
	if l, ok := labels["ru"]; ok && l != "" {
		return l
	}
	return value
}

// CargoStatusDescription returns localized description for cargo status. Fallback: en -> ru.
func CargoStatusDescription(value, lang string) string {
	key := "cargo.cargo_status_desc." + strings.ToUpper(strings.TrimSpace(value))
	lang = strings.ToLower(strings.TrimSpace(lang))
	labels, ok := refLabels[key]
	if !ok {
		return ""
	}
	if l, ok := labels[lang]; ok && l != "" {
		return l
	}
	if l, ok := labels["en"]; ok && l != "" {
		return l
	}
	if l, ok := labels["ru"]; ok && l != "" {
		return l
	}
	return ""
}

var refLabels = map[string]map[string]string{
	"cargo.cargo_status.CREATED":             {"ru": "Создан", "uz": "Yaratilgan", "en": "Created", "tr": "Oluşturuldu", "zh": "已创建"},
	"cargo.cargo_status.PENDING_MODERATION":   {"ru": "На модерации", "uz": "Moderatsiyada", "en": "Pending moderation", "tr": "Moderasyonda", "zh": "待审核"},
	"cargo.cargo_status.SEARCHING":           {"ru": "В поиске перевозчика", "uz": "Tashuvchi qidirilmoqda", "en": "Searching for carrier", "tr": "Taşıyıcı aranıyor", "zh": "寻找承运人"},
	"cargo.cargo_status.REJECTED":             {"ru": "Отклонён", "uz": "Rad etilgan", "en": "Rejected", "tr": "Reddedildi", "zh": "已拒绝"},
	"cargo.cargo_status.ASSIGNED":             {"ru": "Назначен", "uz": "Tayinlangan", "en": "Assigned", "tr": "Atandı", "zh": "已分配"},
	"cargo.cargo_status.IN_PROGRESS":         {"ru": "Выполняется", "uz": "Bajarilmoqda", "en": "In progress", "tr": "Yürütülüyor", "zh": "进行中"},
	"cargo.cargo_status.IN_TRANSIT":          {"ru": "В пути", "uz": "Yo'lda", "en": "In transit", "tr": "Yolda", "zh": "运输中"},
	"cargo.cargo_status.DELIVERED":           {"ru": "Доставлен", "uz": "Yetkazib berilgan", "en": "Delivered", "tr": "Teslim edildi", "zh": "已送达"},
	"cargo.cargo_status.COMPLETED":           {"ru": "Завершён", "uz": "Tugallangan", "en": "Completed", "tr": "Tamamlandı", "zh": "已完成"},
	"cargo.cargo_status.CANCELLED":           {"ru": "Отменён", "uz": "Bekor qilindi", "en": "Cancelled", "tr": "İptal edildi", "zh": "已取消"},
	"cargo.route_point_type.LOAD":    {"ru": "Погрузка", "uz": "Yuklash", "en": "Load", "tr": "Yükleme", "zh": "装货"},
	"cargo.route_point_type.UNLOAD":  {"ru": "Выгрузка", "uz": "Tushirish", "en": "Unload", "tr": "Boşaltma", "zh": "卸货"},
	"cargo.route_point_type.CUSTOMS": {"ru": "Таможня", "uz": "Bojxona", "en": "Customs", "tr": "Gümrük", "zh": "海关"},
	"cargo.route_point_type.TRANSIT": {"ru": "Транзит", "uz": "Tranzit", "en": "Transit", "tr": "Transit", "zh": "过境"},
	"cargo.offer_status.PENDING":     {"ru": "На рассмотрении", "uz": "Ko'rib chiqilmoqda", "en": "Pending", "tr": "Beklemede", "zh": "待处理"},
	"cargo.offer_status.ACCEPTED":    {"ru": "Принят", "uz": "Qabul qilindi", "en": "Accepted", "tr": "Kabul edildi", "zh": "已接受"},
	"cargo.offer_status.REJECTED":    {"ru": "Отклонён", "uz": "Rad etilgan", "en": "Rejected", "tr": "Reddedildi", "zh": "已拒绝"},
	"cargo.created_by_type.ADMIN":      {"ru": "Админ", "uz": "Admin", "en": "Admin", "tr": "Admin", "zh": "管理员"},
	"cargo.created_by_type.DISPATCHER": {"ru": "Диспетчер", "uz": "Dispetcher", "en": "Dispatcher", "tr": "Dispatçı", "zh": "调度员"},
	"cargo.created_by_type.COMPANY":    {"ru": "Компания", "uz": "Kompaniya", "en": "Company", "tr": "Şirket", "zh": "公司"},
	"cargo.truck_type.REFRIGERATOR": {"ru": "Рефрижератор", "uz": "Refrijerator", "en": "Refrigerator", "tr": "Soğutucu", "zh": "冷藏车"},
	"cargo.truck_type.TENT":         {"ru": "Тент", "uz": "Tent", "en": "Tent", "tr": "Tent", "zh": "篷布"},
	"cargo.truck_type.FLATBED":      {"ru": "Борт", "uz": "Bort", "en": "Flatbed", "tr": "Düz platform", "zh": "平板"},
	"cargo.truck_type.TANKER":       {"ru": "Цистерна", "uz": "Sisterna", "en": "Tanker", "tr": "Tanker", "zh": "罐车"},
	"cargo.truck_type.OTHER":        {"ru": "Другое", "uz": "Boshqa", "en": "Other", "tr": "Diğer", "zh": "其他"},
	"cargo.trip_status.PENDING_DRIVER": {"ru": "Ожидание водителя", "uz": "Haydovchi kutilmoqda", "en": "Pending driver", "tr": "Sürücü bekleniyor", "zh": "等待司机"},
	"cargo.trip_status.ASSIGNED":    {"ru": "Назначен", "uz": "Tayinlangan", "en": "Assigned", "tr": "Atandı", "zh": "已分配"},
	"cargo.trip_status.LOADING":     {"ru": "Погрузка", "uz": "Yuklash", "en": "Loading", "tr": "Yükleme", "zh": "装货"},
	"cargo.trip_status.EN_ROUTE":    {"ru": "В пути", "uz": "Yo'lda", "en": "En route", "tr": "Yolda", "zh": "运输中"},
	"cargo.trip_status.UNLOADING":   {"ru": "Выгрузка", "uz": "Tushirish", "en": "Unloading", "tr": "Boşaltma", "zh": "卸货"},
	"cargo.trip_status.COMPLETED":   {"ru": "Завершён", "uz": "Tugallangan", "en": "Completed", "tr": "Tamamlandı", "zh": "已完成"},
	"cargo.trip_status.CANCELLED":   {"ru": "Отменён", "uz": "Bekor qilindi", "en": "Cancelled", "tr": "İptal edildi", "zh": "已取消"},
	"cargo.shipment_type.FTL":     {"ru": "Полная загрузка (FTL)", "uz": "To'liq yuk (FTL)", "en": "Full truck load (FTL)", "tr": "Tam yük (FTL)", "zh": "整车 (FTL)"},
	"cargo.shipment_type.LTL":     {"ru": "Частичная загрузка (LTL)", "uz": "Qisman yuk (LTL)", "en": "Less than truck load (LTL)", "tr": "Kısmi yük (LTL)", "zh": "零担 (LTL)"},
	"cargo.shipment_type.PACKAGE":  {"ru": "Посылка", "uz": "Pochta", "en": "Package", "tr": "Paket", "zh": "包裹"},
	"cargo.shipment_type.OTHER":    {"ru": "Другое", "uz": "Boshqa", "en": "Other", "tr": "Diğer", "zh": "其他"},
	"cargo.currency.USD":   {"ru": "USD (доллар США)", "uz": "USD (AQSh dollari)", "en": "USD (US dollar)", "tr": "USD (ABD doları)", "zh": "USD (美元)"},
	"cargo.currency.UZS":   {"ru": "UZS (сум)", "uz": "UZS (so'm)", "en": "UZS (sum)", "tr": "UZS (sum)", "zh": "UZS (苏姆)"},
	"cargo.currency.EUR":   {"ru": "EUR (евро)", "uz": "EUR (yevro)", "en": "EUR (euro)", "tr": "EUR (euro)", "zh": "EUR (欧元)"},
	"cargo.currency.RUB":   {"ru": "RUB (рубль)", "uz": "RUB (rubl)", "en": "RUB (ruble)", "tr": "RUB (ruble)", "zh": "RUB (卢布)"},
	"cargo.currency.KZT":   {"ru": "KZT (тенге)", "uz": "KZT (tenge)", "en": "KZT (tenge)", "tr": "KZT (tenge)", "zh": "KZT (坚戈)"},
	"cargo.currency.AED":   {"ru": "AED (дирхам ОАЭ)", "uz": "AED (AED dirhami)", "en": "AED (UAE dirham)", "tr": "AED (BAE dirhemi)", "zh": "AED (阿联酋迪拉姆)"},
	"cargo.currency.OTHER": {"ru": "Другая", "uz": "Boshqa", "en": "Other", "tr": "Diğer", "zh": "其他"},
	"cargo.prepayment_type.BANK_TRANSFER": {"ru": "Банковский перевод", "uz": "Bank o'tkazmasi", "en": "Bank transfer", "tr": "Banka havalesi", "zh": "银行转账"},
	"cargo.prepayment_type.CASH":   {"ru": "Наличные", "uz": "Naqd", "en": "Cash", "tr": "Nakit", "zh": "现金"},
	"cargo.prepayment_type.CARD":   {"ru": "Карта", "uz": "Karta", "en": "Card", "tr": "Kart", "zh": "卡"},
	"cargo.prepayment_type.INVOICE": {"ru": "По счёту", "uz": "Hisob-faktura bo'yicha", "en": "Invoice", "tr": "Fatura", "zh": "发票"},
	"cargo.prepayment_type.OTHER":  {"ru": "Другое", "uz": "Boshqa", "en": "Other", "tr": "Diğer", "zh": "其他"},
	"cargo.remaining_type.ON_DELIVERY":   {"ru": "По факту выгрузки", "uz": "Tushirishdan keyin", "en": "On delivery", "tr": "Teslimatta", "zh": "交货时"},
	"cargo.remaining_type.AFTER_INVOICE": {"ru": "По счёту после выгрузки", "uz": "Hisob-faktura tushirishdan keyin", "en": "After invoice post unload", "tr": "Boşaltma sonrası fatura", "zh": "卸货后发票"},
	"cargo.remaining_type.CASH":    {"ru": "Наличными при выгрузке", "uz": "Tushirishda naqd", "en": "Cash on unload", "tr": "Boşaltmada nakit", "zh": "卸货时现金"},
	"cargo.remaining_type.DEFERRED": {"ru": "Отсрочка платежа", "uz": "To'lov muddati", "en": "Deferred payment", "tr": "Ödeme ertelemesi", "zh": "延期付款"},
	"cargo.remaining_type.OTHER":   {"ru": "Другое", "uz": "Boshqa", "en": "Other", "tr": "Diğer", "zh": "其他"},
	"cargo.loading_type.TOP":      {"ru": "Верхняя", "uz": "Yuqori", "en": "Top", "tr": "Üst", "zh": "顶部"},
	"cargo.loading_type.SIDE":     {"ru": "Боковая", "uz": "Yon", "en": "Side", "tr": "Yan", "zh": "侧面"},
	"cargo.loading_type.REAR":     {"ru": "Задняя", "uz": "Orqa", "en": "Rear", "tr": "Arka", "zh": "后部"},
	"cargo.loading_type.CRANE":    {"ru": "Кран", "uz": "Kran", "en": "Crane", "tr": "Vinç", "zh": "起重机"},
	"cargo.loading_type.FORKLIFT": {"ru": "Погрузчик", "uz": "Yuklovchi", "en": "Forklift", "tr": "Forklift", "zh": "叉车"},
	"cargo.loading_type.OTHER":    {"ru": "Другое", "uz": "Boshqa", "en": "Other", "tr": "Diğer", "zh": "其他"},
	// --- drivers ---
	"drivers.registration_step.NAME-OFERTA":    {"ru": "Имя и оферта", "uz": "Ism va oferta", "en": "Name and offer", "tr": "Ad ve teklif", "zh": "姓名和要约"},
	"drivers.registration_step.GEO-PUSH":       {"ru": "Геолокация и push", "uz": "Geolokatsiya va push", "en": "Geolocation and push", "tr": "Konum ve push", "zh": "地理位置和推送"},
	"drivers.registration_step.TRANSPORT-TYPE": {"ru": "Тип транспорта", "uz": "Transport turi", "en": "Transport type", "tr": "Taşıma türü", "zh": "运输类型"},
	"drivers.registration_step.COMPLETED":      {"ru": "Регистрация завершена", "uz": "Ro'yxatdan o'tish tugallandi", "en": "Registration completed", "tr": "Kayıt tamamlandı", "zh": "注册完成"},
	"drivers.registration_status.START": {"ru": "Начало", "uz": "Boshlash", "en": "Start", "tr": "Başlangıç", "zh": "开始"},
	"drivers.registration_status.BASIC": {"ru": "Базовые данные", "uz": "Asosiy ma'lumotlar", "en": "Basic data", "tr": "Temel veriler", "zh": "基本资料"},
	"drivers.registration_status.FULL":  {"ru": "Полная регистрация", "uz": "To'liq ro'yxatdan o'tish", "en": "Full registration", "tr": "Tam kayıt", "zh": "完整注册"},
	"drivers.driver_type.COMPANY":    {"ru": "Компания", "uz": "Kompaniya", "en": "Company", "tr": "Şirket", "zh": "公司"},
	"drivers.driver_type.FREELANCER": {"ru": "Фрилансер", "uz": "Frilanser", "en": "Freelancer", "tr": "Serbest", "zh": "自由职业者"},
	"drivers.driver_type.DRIVER":     {"ru": "Водитель", "uz": "Haydovchi", "en": "Driver", "tr": "Sürücü", "zh": "司机"},
	"drivers.work_status.AVAILABLE": {"ru": "Свободен", "uz": "Mavjud", "en": "Available", "tr": "Müsait", "zh": "可用"},
	"drivers.work_status.LOADED":    {"ru": "Загружен", "uz": "Yuklangan", "en": "Loaded", "tr": "Yüklü", "zh": "已装载"},
	"drivers.work_status.BUSY":      {"ru": "Занят", "uz": "Band", "en": "Busy", "tr": "Meşgul", "zh": "忙碌"},
	"drivers.power_plate.TRUCK":   {"ru": "Грузовик + прицеп", "uz": "Yuk mashinasi + tirkama", "en": "Truck + trailer", "tr": "Kamyon + römork", "zh": "卡车+挂车"},
	"drivers.power_plate.TRACTOR": {"ru": "Тягач + полуприцеп", "uz": "Tyagach + yarim tirkama", "en": "Tractor + semi-trailer", "tr": "Çekici + yarı römork", "zh": "牵引车+半挂车"},
	"drivers.trailer_truck.FLATBED":     {"ru": "Бортовой прицеп", "uz": "Bortli tirkama", "en": "Flatbed trailer", "tr": "Düz platform römork", "zh": "平板挂车"},
	"drivers.trailer_truck.TENTED":      {"ru": "Тентованный прицеп", "uz": "Tentli tirkama", "en": "Tented trailer", "tr": "Tenteli römork", "zh": "篷布挂车"},
	"drivers.trailer_truck.BOX":         {"ru": "Фургонный прицеп", "uz": "Furgon tirkama", "en": "Box trailer", "tr": "Kutu römork", "zh": "厢式挂车"},
	"drivers.trailer_truck.REEFER":      {"ru": "Рефрижераторный прицеп", "uz": "Refrijerator tirkama", "en": "Reefer trailer", "tr": "Soğutucu römork", "zh": "冷藏挂车"},
	"drivers.trailer_truck.TANKER":      {"ru": "Прицеп-цистерна", "uz": "Sisterna tirkama", "en": "Tanker trailer", "tr": "Tanker römork", "zh": "罐车挂车"},
	"drivers.trailer_truck.TIPPER":      {"ru": "Самосвальный прицеп", "uz": "Samosval tirkama", "en": "Tipper trailer", "tr": "Döküm römork", "zh": "自卸挂车"},
	"drivers.trailer_truck.CAR_CARRIER": {"ru": "Прицеп-автовоз", "uz": "Avtovoz tirkama", "en": "Car carrier trailer", "tr": "Araç taşıyıcı römork", "zh": "车辆运输挂车"},
	"drivers.trailer_tractor.FLATBED":   {"ru": "Бортовой полуприцеп", "uz": "Bortli yarim tirkama", "en": "Flatbed semi-trailer", "tr": "Düz platform yarı römork", "zh": "平板半挂车"},
	"drivers.trailer_tractor.TENTED":    {"ru": "Тентованный полуприцеп", "uz": "Tentli yarim tirkama", "en": "Tented semi-trailer", "tr": "Tenteli yarı römork", "zh": "篷布半挂车"},
	"drivers.trailer_tractor.BOX":       {"ru": "Фургонный полуприцеп", "uz": "Furgon yarim tirkama", "en": "Box semi-trailer", "tr": "Kutu yarı römork", "zh": "厢式半挂车"},
	"drivers.trailer_tractor.REEFER":   {"ru": "Рефрижераторный полуприцеп", "uz": "Refrijerator yarim tirkama", "en": "Reefer semi-trailer", "tr": "Soğutucu yarı römork", "zh": "冷藏半挂车"},
	"drivers.trailer_tractor.TANKER":   {"ru": "Полуприцеп-цистерна", "uz": "Sisterna yarim tirkama", "en": "Tanker semi-trailer", "tr": "Tanker yarı römork", "zh": "罐车半挂车"},
	"drivers.trailer_tractor.LOWBED":   {"ru": "Низкорамный полуприцеп", "uz": "Past ramkali yarim tirkama", "en": "Lowbed semi-trailer", "tr": "Alçak yarı römork", "zh": "低平板半挂车"},
	"drivers.trailer_tractor.CONTAINER": {"ru": "Полуприцеп-контейнеровоз", "uz": "Konteyner yarim tirkama", "en": "Container semi-trailer", "tr": "Konteyner yarı römork", "zh": "集装箱半挂车"},
	// --- admin ---
	"admin.admin_status.ACTIVE":   {"ru": "Активен", "uz": "Faol", "en": "Active", "tr": "Aktif", "zh": "活跃"},
	"admin.admin_status.INACTIVE": {"ru": "Неактивен", "uz": "Nofaol", "en": "Inactive", "tr": "Pasif", "zh": "未激活"},
	"admin.admin_status.BLOCKED":  {"ru": "Заблокирован", "uz": "Bloklangan", "en": "Blocked", "tr": "Engelli", "zh": "已封锁"},
	"admin.admin_type.CREATOR":   {"ru": "Создатель", "uz": "Yaratuvchi", "en": "Creator", "tr": "Oluşturan", "zh": "创建者"},
	"admin.admin_type.OPERATOR":  {"ru": "Оператор", "uz": "Operator", "en": "Operator", "tr": "Operatör", "zh": "操作员"},
	// --- dispatchers ---
	"dispatchers.work_status.AVAILABLE": {"ru": "Доступен", "uz": "Mavjud", "en": "Available", "tr": "Müsait", "zh": "可用"},
	"dispatchers.work_status.BUSY":     {"ru": "Занят", "uz": "Band", "en": "Busy", "tr": "Meşgul", "zh": "忙碌"},
	"dispatchers.work_status.OFFLINE":  {"ru": "Не в сети", "uz": "Oflayn", "en": "Offline", "tr": "Çevrimdışı", "zh": "离线"},
	// --- company ---
	"company.company_type.SHIPPER": {"ru": "Грузоотправитель", "uz": "Yuk jo'natuvchi", "en": "Shipper", "tr": "Gönderici", "zh": "发货人"},
	"company.company_type.CARRIER": {"ru": "Перевозчик", "uz": "Tashuvchi", "en": "Carrier", "tr": "Taşıyıcı", "zh": "承运人"},
	"company.company_type.BROKER":  {"ru": "Брокер", "uz": "Broker", "en": "Broker", "tr": "Broker", "zh": "经纪人"},
	"company.company_status.ACTIVE":   {"ru": "Активна", "uz": "Faol", "en": "Active", "tr": "Aktif", "zh": "活跃"},
	"company.company_status.INACTIVE": {"ru": "Неактивна", "uz": "Nofaol", "en": "Inactive", "tr": "Pasif", "zh": "未激活"},
	"company.company_status.BLOCKED":  {"ru": "Заблокирована", "uz": "Bloklangan", "en": "Blocked", "tr": "Engelli", "zh": "已封锁"},
	"company.company_status.PENDING":  {"ru": "На модерации", "uz": "Moderatsiyada", "en": "Pending", "tr": "Beklemede", "zh": "待审核"},
	"company.role.OWNER":          {"ru": "Владелец", "uz": "Egasi", "en": "Owner", "tr": "Sahip", "zh": "所有者"},
	"company.role.CEO":            {"ru": "Директор", "uz": "Direktor", "en": "CEO", "tr": "CEO", "zh": "首席执行官"},
	"company.role.TOP_MANAGER":    {"ru": "Старший менеджер", "uz": "Katta menejer", "en": "Top manager", "tr": "Üst düzey yönetici", "zh": "高级经理"},
	"company.role.TOP_DISPATCHER": {"ru": "Старший диспетчер", "uz": "Katta dispetcher", "en": "Top dispatcher", "tr": "Kıdemli dispatçı", "zh": "高级调度员"},
	"company.role.DISPATCHER":     {"ru": "Диспетчер", "uz": "Dispetcher", "en": "Dispatcher", "tr": "Dispatçı", "zh": "调度员"},
	"company.role.MANAGER":        {"ru": "Менеджер", "uz": "Menejer", "en": "Manager", "tr": "Yönetici", "zh": "经理"},
	// --- transport (driver transport-options) ---
	"transport.power_plate.TRUCK":   {"ru": "Грузовик + прицеп", "uz": "Yuk mashinasi + tirkama", "en": "Truck + trailer", "tr": "Kamyon + römork", "zh": "卡车+挂车"},
	"transport.power_plate.TRACTOR": {"ru": "Тягач + полуприцеп", "uz": "Tyagach + yarim tirkama", "en": "Tractor + semi-trailer", "tr": "Çekici + yarı römork", "zh": "牵引车+半挂车"},
	"transport.trailer_truck.FLATBED":     {"ru": "Бортовой прицеп", "uz": "Bortli tirkama", "en": "Flatbed trailer", "tr": "Düz platform römork", "zh": "平板挂车"},
	"transport.trailer_truck.TENTED":      {"ru": "Тентованный прицеп", "uz": "Tentli tirkama", "en": "Tented trailer", "tr": "Tenteli römork", "zh": "篷布挂车"},
	"transport.trailer_truck.BOX":         {"ru": "Фургонный прицеп", "uz": "Furgon tirkama", "en": "Box trailer", "tr": "Kutu römork", "zh": "厢式挂车"},
	"transport.trailer_truck.REEFER":      {"ru": "Рефрижераторный прицеп", "uz": "Refrijerator tirkama", "en": "Reefer trailer", "tr": "Soğutucu römork", "zh": "冷藏挂车"},
	"transport.trailer_truck.TANKER":      {"ru": "Прицеп-цистерна", "uz": "Sisterna tirkama", "en": "Tanker trailer", "tr": "Tanker römork", "zh": "罐车挂车"},
	"transport.trailer_truck.TIPPER":      {"ru": "Самосвальный прицеп", "uz": "Samosval tirkama", "en": "Tipper trailer", "tr": "Döküm römork", "zh": "自卸挂车"},
	"transport.trailer_truck.CAR_CARRIER": {"ru": "Прицеп-автовоз", "uz": "Avtovoz tirkama", "en": "Car carrier trailer", "tr": "Araç taşıyıcı römork", "zh": "车辆运输挂车"},
	"transport.trailer_tractor.FLATBED":   {"ru": "Бортовой полуприцеп", "uz": "Bortli yarim tirkama", "en": "Flatbed semi-trailer", "tr": "Düz platform yarı römork", "zh": "平板半挂车"},
	"transport.trailer_tractor.TENTED":    {"ru": "Тентованный полуприцеп", "uz": "Tentli yarim tirkama", "en": "Tented semi-trailer", "tr": "Tenteli yarı römork", "zh": "篷布半挂车"},
	"transport.trailer_tractor.BOX":       {"ru": "Фургонный полуприцеп", "uz": "Furgon yarim tirkama", "en": "Box semi-trailer", "tr": "Kutu yarı römork", "zh": "厢式半挂车"},
	"transport.trailer_tractor.REEFER":   {"ru": "Рефрижераторный полуприцеп", "uz": "Refrijerator yarim tirkama", "en": "Reefer semi-trailer", "tr": "Soğutucu yarı römork", "zh": "冷藏半挂车"},
	"transport.trailer_tractor.TANKER":   {"ru": "Полуприцеп-цистерна", "uz": "Sisterna yarim tirkama", "en": "Tanker semi-trailer", "tr": "Tanker yarı römork", "zh": "罐车半挂车"},
	"transport.trailer_tractor.LOWBED":   {"ru": "Низкорамный полуприцеп", "uz": "Past ramkali yarim tirkama", "en": "Lowbed semi-trailer", "tr": "Alçak yarı römork", "zh": "低平板半挂车"},
	"transport.trailer_tractor.CONTAINER": {"ru": "Полуприцеп-контейнеровоз", "uz": "Konteyner yarim tirkama", "en": "Container semi-trailer", "tr": "Konteyner yarı römork", "zh": "集装箱半挂车"},
}

func init() {
	descs := map[string]map[string]string{
		"cargo.cargo_status_desc.CREATED":             {"ru": "Груз создан.", "uz": "Yuk yaratilgan.", "en": "Cargo created.", "tr": "Kargo oluşturuldu.", "zh": "货物已创建。"},
		"cargo.cargo_status_desc.PENDING_MODERATION":  {"ru": "На модерации; админ примет (searching) или отклонит (rejected).", "uz": "Moderatsiyada; admin qabul (searching) yoki rad (rejected) qiladi.", "en": "Pending moderation; admin will accept (searching) or reject (rejected).", "tr": "Moderasyonda; admin kabul (searching) veya red (rejected) eder.", "zh": "待审核；管理员通过(searching)或拒绝(rejected)。"},
		"cargo.cargo_status_desc.SEARCHING":           {"ru": "Груз виден водителям; принимаются офферы.", "uz": "Yuk haydovchilarga ko'rinadi; takliflar qabul qilinadi.", "en": "Cargo visible to drivers; offers accepted.", "tr": "Yük sürücülere görünür; teklifler kabul edilir.", "zh": "货物对司机可见；接受报价。"},
		"cargo.cargo_status_desc.REJECTED":           {"ru": "Отклонён модерацией; указана причина.", "uz": "Moderatsiya tomonidan rad etilgan; sabab ko'rsatilgan.", "en": "Rejected by moderation; reason provided.", "tr": "Moderasyon tarafından reddedildi; neden belirtildi.", "zh": "审核拒绝；已提供原因。"},
		"cargo.cargo_status_desc.ASSIGNED":            {"ru": "Перевозчик выбран; ожидается погрузка.", "uz": "Tashuvchi tanlandi; yuklash kutilmoqda.", "en": "Carrier selected; loading expected.", "tr": "Taşıyıcı seçildi; yükleme bekleniyor.", "zh": "已选承运人；等待装货。"},
		"cargo.cargo_status_desc.IN_PROGRESS":         {"ru": "Выполняется; водитель в пути или грузит.", "uz": "Bajarilmoqda; haydovchi yo'lda yoki yuklayapti.", "en": "In progress; driver en route or loading.", "tr": "Yürütülüyor; sürücü yolda veya yüklüyor.", "zh": "进行中；司机在途或装货。"},
		"cargo.cargo_status_desc.IN_TRANSIT":         {"ru": "Груз в перевозке.", "uz": "Yuk tashilmoqda.", "en": "Cargo in transit.", "tr": "Yük taşınıyor.", "zh": "货物运输中。"},
		"cargo.cargo_status_desc.DELIVERED":          {"ru": "Доставлен.", "uz": "Yetkazib berildi.", "en": "Delivered.", "tr": "Teslim edildi.", "zh": "已送达。"},
		"cargo.cargo_status_desc.COMPLETED":          {"ru": "Перевозка завершена.", "uz": "Tashish tugallandi.", "en": "Shipment completed.", "tr": "Taşıma tamamlandı.", "zh": "运输已完成。"},
		"cargo.cargo_status_desc.CANCELLED":          {"ru": "Груз отменён.", "uz": "Yuk bekor qilindi.", "en": "Cargo cancelled.", "tr": "Yük iptal edildi.", "zh": "货物已取消。"},
	}
	for k, v := range descs {
		refLabels[k] = v
	}
}
