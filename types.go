// Package geoutil provides advanced geospatial utilities with optimized concurrent processing.
package geoutil

import "time"

// Point represents a geographic coordinate
type Point struct {
    Lat float64 `json:"lat"` // Latitude in degrees (-90 to 90)
    Lon float64 `json:"lon"` // Longitude in degrees (-180 to 180)
}

// Location contains comprehensive geographic information
type Location struct {
    Country   string  `json:"country"`   // Country name
    City      string  `json:"city"`      // City name
    Address   string  `json:"address"`   // Full address
    Lat       float64 `json:"lat"`       // Latitude
    Lon       float64 `json:"lon"`       // Longitude
    Elevation int     `json:"elevation"` // Elevation in meters
    Timezone  string  `json:"timezone"`  // IANA timezone identifier
}

// GeocoderConfig defines settings for geocoding services
type GeocoderConfig struct {
    UserAgent      string        `json:"user_agent"`      // Required User-Agent header for APIs
    RequestsPerSec int           `json:"requests_per_sec"` // Request rate limit (requests/second)
    Timeout        time.Duration `json:"timeout"`         // Request timeout duration
}

// Geocoder interface defines geocoding operations
type Geocoder interface {
    Geocode(address string) (Point, error)                   // Convert address to coordinates
    ReverseGeocode(p Point) (Location, error)                // Convert coordinates to address
    BatchGeocode(addresses []string) ([]Point, error)        // Batch address processing
    BatchReverseGeocode(points []Point) ([]Location, error)  // Batch reverse geocoding
}

// ElevationProvider interface defines elevation data operations
type ElevationProvider interface {
    GetElevation(p Point) (int, error)              // Get elevation for single point
    BatchGetElevation(points []Point) ([]int, error) // Batch elevation processing
}