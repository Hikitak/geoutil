package geoutil

import (
	"time"
)

// Point - географическая точка
type Point struct {
	Lat float64 `json:"lat"` // Широта (-90 до 90)
	Lon float64 `json:"lon"` // Долгота (-180 до 180)
}

// Location - полная географическая информация
type Location struct {
	Country   string  `json:"country"`   // Страна
	City      string  `json:"city"`      // Город
	Address   string  `json:"address"`   // Адрес
	Lat       float64 `json:"lat"`       // Широта
	Lon       float64 `json:"lon"`       // Долгота
	Elevation int     `json:"elevation"` // Высота (метры)
	Timezone  string  `json:"timezone"`  // Временная зона
}

// GeocoderConfig - конфигурация геокодера
type GeocoderConfig struct {
	UserAgent      string        `json:"user_agent"`     // User-Agent для запросов
	RequestsPerSec int           `json:"requests_per_sec"` // Лимит запросов/сек
	Timeout        time.Duration `json:"timeout"`        // Таймаут запросов
}

// Geocoder - интерфейс геокодирования
type Geocoder interface {
	Geocode(address string) (Point, error)
	ReverseGeocode(p Point) (Location, error)
	BatchGeocode(addresses []string) ([]Point, error)
	BatchReverseGeocode(points []Point) ([]Location, error)
}

// ElevationProvider - интерфейс работы с высотами
type ElevationProvider interface {
	GetElevation(p Point) (int, error)
	BatchGetElevation(points []Point) ([]int, error)
}