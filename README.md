# Geoutil - Advanced Geospatial Utilities for Go

Geoutil - это высокопроизводительный пакет для работы с геопространственными данными в Go, с оптимизированной поддержкой конкурентных операций. Пакет предоставляет:

- Геокодирование и обратное геокодирование (через OSM Nominatim)
- Получение данных о высоте (через Open-Elevation API)
- Расчёты расстояний (Haversine)
- Геометрические операции (точка в полигоне)
- Пакетную обработку с автоматическим ограничением запросов
- Кеширование результатов

## Установка

```bash
go get github.com/yourusername/geoutil
```

## Основные функции
### Геокодирование
```go
config := geoutil.GeocoderConfig{
    UserAgent: "MyApp/1.0",
    RequestsPerSec: 2,
}
geocoder := geoutil.NewNominatimGeocoder(config)

// Одиночное геокодирование
point, _ := geocoder.Geocode("Red Square, Moscow")

// Пакетное геокодирование
addresses := []string{"Paris", "Berlin", "London"}
points, _ := geocoder.BatchGeocode(addresses)
```

### Получение высот
```go
elevation := geoutil.NewOpenElevationProvider(5) // 5 запросов/сек

// Одиночный запрос
height, _ := elevation.GetElevation(geoutil.Point{55.75, 37.61})

// Пакетная обработка
points := []geoutil.Point{{55.75, 37.61}, {59.93, 30.31}}
heights, _ := elevation.BatchGetElevation(points)
```
### Расчёты расстояний
```go
moscow := geoutil.Point{55.7558, 37.6173}
newYork := geoutil.Point{40.7128, -74.0060}

// Расстояние между двумя точками
distance := geoutil.DistanceHaversine(moscow, newYork) // ~7500 км

// Матрица расстояний
points := []geoutil.Point{moscow, newYork, {51.5074, -0.1278}}
matrix := geoutil.BatchDistanceConcurrent(points, geoutil.DistanceHaversine)
```
### Геометрические операции
```go
// Определение полигона Москвы
polygon := []geoutil.Point{
    {55.113, 36.8}, {56.021, 37.967}, 
    {56.047, 37.806}, {55.631, 37.332},
}

// Проверка точки в полигоне
inside := geoutil.IsPointInPolygon(geoutil.Point{55.75, 37.61}, polygon) // true

// Фильтрация точек
points := []geoutil.Point{
    {55.75, 37.61}, {40.71, -74.00}, {55.60, 37.50},
}
filtered := geoutil.FilterPointsInPolygonConcurrent(points, polygon)
```
### Получение полной информации
```go
// Полный геопрофиль точки
location, _ := geoutil.FullLocation(
    geoutil.Point{55.7558, 37.6173},
    geocoder,
    elevation,
)

// Пакетная обработка
points := []geoutil.Point{
    {55.7558, 37.6173}, // Москва
    {40.7128, -74.0060}, // Нью-Йорк
}
locations, _ := geoutil.BatchFullLocation(points, geocoder, elevation)
```