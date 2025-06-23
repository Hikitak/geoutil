package geoutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// NominatimGeocoder - реализация Geocoder для OSM Nominatim
type NominatimGeocoder struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	cache      *Cache
	config     GeocoderConfig
}

// NewNominatimGeocoder создает новый геокодер
func NewNominatimGeocoder(config GeocoderConfig) *NominatimGeocoder {
	if config.RequestsPerSec == 0 {
		config.RequestsPerSec = 1
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &NominatimGeocoder{
		baseURL: "https://nominatim.openstreetmap.org",
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		limiter: rate.NewLimiter(rate.Limit(config.RequestsPerSec), 1),
		cache:   NewCache(24 * time.Hour),
		config:  config,
	}
}

// Geocode преобразует адрес в координаты
func (n *NominatimGeocoder) Geocode(address string) (Point, error) {
	// Проверка кеша
	if val, found := n.cache.Get(address); found {
		return val.(Point), nil
	}

	// Ограничение частоты запросов
	ctx, cancel := context.WithTimeout(context.Background(), n.config.Timeout)
	defer cancel()
	if err := n.limiter.Wait(ctx); err != nil {
		return Point{}, err
	}

	// Формирование запроса
	params := url.Values{
		"q":      {address},
		"format": {"json"},
		"limit":  {"1"},
	}
	url := fmt.Sprintf("%s/search?%s", n.baseURL, params.Encode())

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", n.config.UserAgent)

	// Выполнение запроса
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return Point{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Point{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Парсинг ответа
	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return Point{}, err
	}

	if len(results) == 0 {
		return Point{}, errors.New("address not found")
	}

	// Конвертация координат
	lat, _ := strconv.ParseFloat(results[0].Lat, 64)
	lon, _ := strconv.ParseFloat(results[0].Lon, 64)
	point := Point{Lat: lat, Lon: lon}

	// Сохранение в кеш
	n.cache.Set(address, point)
	return point, nil
}

// BatchGeocode конкурентное геокодирование адресов
func (n *NominatimGeocoder) BatchGeocode(addresses []string) ([]Point, error) {
	type result struct {
		index int
		point Point
		err   error
	}

	results := make(chan result, len(addresses))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Ограничение одновременных запросов

	for i, addr := range addresses {
		wg.Add(1)
		go func(idx int, address string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			point, err := n.Geocode(address)
			results <- result{idx, point, err}
		}(i, addr)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	points := make([]Point, len(addresses))
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		points[res.index] = res.point
	}

	return points, nil
}

// ReverseGeocode преобразует координаты в адрес
func (n *NominatimGeocoder) ReverseGeocode(p Point) (Location, error) {
	cacheKey := fmt.Sprintf("reverse_%f_%f", p.Lat, p.Lon)
	if val, found := n.cache.Get(cacheKey); found {
		return val.(Location), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), n.config.Timeout)
	defer cancel()
	if err := n.limiter.Wait(ctx); err != nil {
		return Location{}, err
	}

	params := url.Values{
		"lat":    {fmt.Sprintf("%f", p.Lat)},
		"lon":    {fmt.Sprintf("%f", p.Lon)},
		"format": {"json"},
	}
	url := fmt.Sprintf("%s/reverse?%s", n.baseURL, params.Encode())

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", n.config.UserAgent)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return Location{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Location{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var data struct {
		Address struct {
			Country   string `json:"country"`
			City      string `json:"city"`
			Road      string `json:"road"`
			House     string `json:"house_number"`
			Postcode  string `json:"postcode"`
		} `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return Location{}, err
	}

	loc := Location{
		Country: data.Address.Country,
		City:    data.Address.City,
		Address: fmt.Sprintf("%s %s", data.Address.Road, data.Address.House),
		Lat:     p.Lat,
		Lon:     p.Lon,
	}

	n.cache.Set(cacheKey, loc)
	return loc, nil
}