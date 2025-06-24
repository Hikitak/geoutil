# Geoutil - Advanced Geospatial Utilities for Go

Geoutil is a high-performance Go package for geospatial operations with optimized concurrent processing. It provides:

- Geocoding and reverse geocoding (OSM Nominatim)
- Elevation data (Open-Elevation API)
- Distance calculations (Haversine formula)
- Point-in-polygon filtering
- Batch processing with automatic rate limiting
- Comprehensive caching and error handling

## Installation

```bash
go get github.com/Hikitak/geoutil
```

## Quick Start
### Geocoding
```go
package main

import (
	"fmt"
	"time"
	
	"github.com/yourusername/geoutil"
)

func main() {
	config := geoutil.GeocoderConfig{
		UserAgent:      "MyApp/1.0",
		RequestsPerSec: 2,
		Timeout:        15 * time.Second,
	}
	gc := geoutil.NewNominatimGeocoder(config)
	
	// Single geocode
	point, _ := gc.Geocode("Eiffel Tower, Paris")
	fmt.Printf("Coordinates: %.4f, %.4f\n", point.Lat, point.Lon)
	
	// Batch geocode
	addresses := []string{"London", "Berlin", "Madrid"}
	points, _ := gc.BatchGeocode(addresses)
	for i, p := range points {
		fmt.Printf("%s: %.4f, %.4f\n", addresses[i], p.Lat, p.Lon)
	}
}
```

### Distance Calculations

```go
func main() {
	moscow := geoutil.Point{55.7558, 37.6173}
	newYork := geoutil.Point{40.7128, -74.0060}
	
	// Single distance
	distance := geoutil.DistanceHaversine(moscow, newYork)
	fmt.Printf("Distance: %.2f km\n", distance)
	
	// Distance matrix
	points := []geoutil.Point{
		{55.7558, 37.6173}, // Moscow
		{40.7128, -74.0060}, // New York
		{51.5074, -0.1278}, // London
	}
	matrix := geoutil.BatchDistanceConcurrent(points, geoutil.DistanceHaversine)
	fmt.Println("Distance matrix:", matrix)
}
```
### Point-in-Polygon Filtering

```go
func main() {
	// Define polygon (London approximate boundaries)
	polygon := []geoutil.Point{
		{51.28, -0.50}, {51.68, -0.50},
		{51.68, 0.25}, {51.28, 0.25},
	}
	
	// Generate random points
	points := make([]geoutil.Point, 10_000)
	for i := range points {
		points[i] = geoutil.Point{
			Lat: 51.3 + rand.Float64()*0.4,
			Lon: -0.45 + rand.Float64()*0.7,
		}
	}
	
	// Filter points in polygon
	filtered := geoutil.FilterPointsInPolygonConcurrent(points, polygon)
	fmt.Printf("Points inside polygon: %d/%d\n", len(filtered), len(points))
}
```

## API Documentation
### Types
```go
type Point struct {
    Lat float64 // Latitude [-90, 90]
    Lon float64 // Longitude [-180, 180]
}

type Location struct {
    Country   string  // Country name
    City      string  // City name
    Address   string  // Full address
    Lat       float64 // Latitude
    Lon       float64 // Longitude
    Elevation int     // Elevation in meters
    Timezone  string  // IANA timezone
}
```
### Core Functions

```go
// Geocoding
func NewNominatimGeocoder(config GeocoderConfig) *NominatimGeocoder
func (n *NominatimGeocoder) Geocode(address string) (Point, error)
func (n *NominatimGeocoder) BatchGeocode(addresses []string) ([]Point, error)

// Elevation
func NewOpenElevationProvider(rps int) *OpenElevationProvider
func (o *OpenElevationProvider) GetElevation(p Point) (int, error)
func (o *OpenElevationProvider) BatchGetElevation(points []Point) ([]int, error)

// Distance
func DistanceHaversine(p1, p2 Point) float64
func BatchDistanceConcurrent(points []Point, distanceFunc func(p1, p2 Point) float64) [][]float64

// Geometry
func IsPointInPolygon(p Point, polygon []Point) bool
func FilterPointsInPolygonConcurrent(points []Point, polygon []Point) []Point

// Comprehensive Data
func FullLocation(p Point, geocoder Geocoder, elevation ElevationProvider) (Location, error)
func BatchFullLocation(points []Point, geocoder Geocoder, elevation ElevationProvider) ([]Location, error)
```

## Configuration Tips
1. User-Agent: Always set a meaningful User-Agent in GeocoderConfig

2. Rate Limits:
    - Nominatim: 1 req/sec (default), increase only if you have permission

    - Open-Elevation: 5-10 req/sec recommended

3. Caching:

    - Geocoding results cached for 24 hours

    - Elevation data cached for 30 days

4. Concurrency:

    - I/O-bound operations (geocoding/elevation): Limit to 10-20 concurrent

    - CPU-bound operations (distance/geometry): Use NumCPU() workers

## Limitations
- Timezone support requires external library

- Vincenty formula not implemented (use Haversine for most cases)

- Requires Go 1.18+ for generics in cache implementation