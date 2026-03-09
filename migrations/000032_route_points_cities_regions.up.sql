-- Справочники городов (код: TAS, SAM, DXB) и регионов (области). Точки маршрута: city_code, region_code, orientir.

CREATE TABLE IF NOT EXISTS cities (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code VARCHAR(20) NOT NULL UNIQUE,
  name_ru VARCHAR(255) NOT NULL,
  name_en VARCHAR(255) NULL,
  country_code VARCHAR(3) NOT NULL,
  lat DOUBLE PRECISION NULL,
  lng DOUBLE PRECISION NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS regions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code VARCHAR(20) NOT NULL,
  name_ru VARCHAR(255) NOT NULL,
  name_en VARCHAR(255) NULL,
  country_code VARCHAR(3) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(country_code, code)
);

CREATE INDEX IF NOT EXISTS idx_cities_country ON cities (country_code);
CREATE INDEX IF NOT EXISTS idx_regions_country ON regions (country_code);

ALTER TABLE route_points ADD COLUMN IF NOT EXISTS city_code VARCHAR(20) NULL;
ALTER TABLE route_points ADD COLUMN IF NOT EXISTS region_code VARCHAR(20) NULL;
ALTER TABLE route_points ADD COLUMN IF NOT EXISTS orientir VARCHAR(500) NULL;

-- Начальные данные: города (коды TAS, SAM, DXB и др.) и регионы (области Узбекистана и др.)
INSERT INTO cities (code, name_ru, name_en, country_code, lat, lng) VALUES
  ('TAS', 'Ташкент', 'Tashkent', 'UZ', 41.311081, 69.240562),
  ('SAM', 'Самарканд', 'Samarkand', 'UZ', 39.654167, 66.959722),
  ('AND', 'Андижан', 'Andijan', 'UZ', 40.783333, 72.333333),
  ('BUK', 'Бухара', 'Bukhara', 'UZ', 39.774722, 64.428611),
  ('FER', 'Фергана', 'Fergana', 'UZ', 40.386389, 71.786389),
  ('NAM', 'Наманган', 'Namangan', 'UZ', 40.998333, 71.672778),
  ('NUK', 'Нукус', 'Nukus', 'UZ', 42.453056, 59.610278),
  ('QAR', 'Карши', 'Qarshi', 'UZ', 38.861111, 65.789167),
  ('TER', 'Термез', 'Termez', 'UZ', 37.224167, 67.278333),
  ('DXB', 'Дубай', 'Dubai', 'AE', 25.204849, 55.270783),
  ('AUH', 'Абу-Даби', 'Abu Dhabi', 'AE', 24.453889, 54.377343),
  ('SHJ', 'Шарджа', 'Sharjah', 'AE', 25.357308, 55.403304),
  ('ALA', 'Алматы', 'Almaty', 'KZ', 43.238949, 76.945465),
  ('TSE', 'Астана', 'Astana', 'KZ', 51.160523, 71.470356),
  ('FRU', 'Бишкек', 'Bishkek', 'KG', 42.874722, 74.612222),
  ('DUS', 'Душанбе', 'Dushanbe', 'TJ', 38.559772, 68.773928),
  ('ASB', 'Ашхабад', 'Ashgabat', 'TM', 37.950000, 58.383333),
  ('MOW', 'Москва', 'Moscow', 'RU', 55.755826, 37.617299),
  ('LED', 'Санкт-Петербург', 'Saint Petersburg', 'RU', 59.934280, 30.335099),
  ('IST', 'Стамбул', 'Istanbul', 'TR', 41.008238, 28.978359),
  ('TEH', 'Тегеран', 'Tehran', 'IR', 35.689252, 51.389600),
  ('DEL', 'Дели', 'Delhi', 'IN', 28.613939, 77.209021),
  ('PEK', 'Пекин', 'Beijing', 'CN', 39.904200, 116.407396)
ON CONFLICT (code) DO NOTHING;

INSERT INTO regions (code, name_ru, name_en, country_code) VALUES
  ('TAS', 'г. Ташкент', 'Tashkent City', 'UZ'),
  ('TK', 'Ташкентская область', 'Tashkent Region', 'UZ'),
  ('SA', 'Самаркандская область', 'Samarkand Region', 'UZ'),
  ('AN', 'Андижанская область', 'Andijan Region', 'UZ'),
  ('BU', 'Бухарская область', 'Bukhara Region', 'UZ'),
  ('FA', 'Ферганская область', 'Fergana Region', 'UZ'),
  ('NG', 'Наманганская область', 'Namangan Region', 'UZ'),
  ('QR', 'Каракалпакстан', 'Karakalpakstan', 'UZ'),
  ('QA', 'Кашкадарьинская область', 'Kashkadarya Region', 'UZ'),
  ('SU', 'Сурхандарьинская область', 'Surkhandarya Region', 'UZ'),
  ('XO', 'Хорезмская область', 'Khorezm Region', 'UZ'),
  ('JI', 'Джизакская область', 'Jizzakh Region', 'UZ'),
  ('SI', 'Сырдарьинская область', 'Sirdarya Region', 'UZ'),
  ('NV', 'Навоийская область', 'Navoiy Region', 'UZ'),
  ('DXB', 'Дубай', 'Dubai', 'AE'),
  ('AUH', 'Абу-Даби', 'Abu Dhabi', 'AE'),
  ('SHJ', 'Шарджа', 'Sharjah', 'AE'),
  ('AZ', 'Аджман', 'Ajman', 'AE'),
  ('RK', 'Рас-эль-Хайма', 'Ras Al Khaimah', 'AE'),
  ('FU', 'Фуджейра', 'Fujairah', 'AE'),
  ('UQ', 'Умм-эль-Кайвайн', 'Umm Al Quwain', 'AE')
ON CONFLICT (country_code, code) DO NOTHING;
