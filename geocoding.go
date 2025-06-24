package geoutil

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "strconv"
    "time"

    "golang.org/x/time/rate"
)

// NominatimGeocoder implements Geocoder using OSM Nominatim service
type NominatimGeocoder struct {
    baseURL    string
    httpClient *http.Client
    limiter    *rate.Limiter
    cache      *Cache
    config     GeocoderConfig
}

// NewNominatimGeocoder creates a new Nominatim geocoder
// config: Configuration parameters
func NewNominatimGeocoder(config GeocoderConfig) *NominatimGeocoder {
    // Set default values
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

// Geocode converts address to geographic coordinates
// address: Human-readable address string
// Returns: Geographic point or error
func (n *NominatimGeocoder) Geocode(address string) (Point, error) {
    // Check cache first
    if val, found := n.cache.Get(address); found {
        return val.(Point), nil
    }

    // Apply rate limiting
    ctx, cancel := context.WithTimeout(context.Background(), n.config.Timeout)
    defer cancel()
    if err := n.limiter.Wait(ctx); err != nil {
        return Point{}, err
    }

    // Build request URL
    params := url.Values{
        "q":      {address},
        "format": {"json"},
        "limit":  {"1"},
    }
    url := fmt.Sprintf("%s/search?%s", n.baseURL, params.Encode())

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return Point{}, err
    }
    req.Header.Set("User-Agent", n.config.UserAgent)

    // Execute request
    resp, err := n.httpClient.Do(req)
    if err != nil {
        return Point{}, err
    }
    defer resp.Body.Close()

    // Check status code
    if resp.StatusCode != http.StatusOK {
        return Point{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
    }

    // Parse response
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

    // Convert coordinates
    lat, err := strconv.ParseFloat(results[0].Lat, 64)
    if err != nil {
        return Point{}, err
    }
    lon, err := strconv.ParseFloat(results[0].Lon, 64)
    if err != nil {
        return Point{}, err
    }
    point := Point{Lat: lat, Lon: lon}

    // Cache result
    n.cache.Set(address, point)
    return point, nil
}

// BatchGeocode processes multiple addresses concurrently
// addresses: Slice of address strings
// Returns: Slice of points or first error encountered
func (n *NominatimGeocoder) BatchGeocode(addresses []string) ([]Point, error) {
    type result struct {
        index int
        point Point
        err   error
    }

    results := make(chan result, len(addresses))
    var wg sync.WaitGroup
    sem := make(chan struct{}, 10) // Concurrency limiter

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

    // Close results channel when all workers complete
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    points := make([]Point, len(addresses))
    for res := range results {
        if res.err != nil {
            return nil, res.err
        }
        points[res.index] = res.point
    }

    return points, nil
}

// ReverseGeocode converts coordinates to address information
// p: Geographic point
// Returns: Location details or error
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

    // Build request URL
    params := url.Values{
        "lat":    {fmt.Sprintf("%f", p.Lat)},
        "lon":    {fmt.Sprintf("%f", p.Lon)},
        "format": {"json"},
    }
    url := fmt.Sprintf("%s/reverse?%s", n.baseURL, params.Encode())

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return Location{}, err
    }
    req.Header.Set("User-Agent", n.config.UserAgent)

    resp, err := n.httpClient.Do(req)
    if err != nil {
        return Location{}, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return Location{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
    }

    // Parse response
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