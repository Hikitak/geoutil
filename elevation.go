package geoutil

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "math"
    "net/http"
    "strings"
    "time"

    "golang.org/x/time/rate"
)

// OpenElevationProvider implements ElevationProvider using Open-Elevation API
type OpenElevationProvider struct {
    baseURL    string
    httpClient *http.Client
    limiter    *rate.Limiter
    cache      *Cache
}

// NewOpenElevationProvider creates an elevation provider instance
// rps: Requests per second limit
func NewOpenElevationProvider(rps int) *OpenElevationProvider {
    if rps == 0 {
        rps = 5
    }
    
    return &OpenElevationProvider{
        baseURL:    "https://api.open-elevation.com/api/v1/lookup",
        httpClient: &http.Client{Timeout: 10 * time.Second},
        limiter:    rate.NewLimiter(rate.Limit(rps), 1),
        cache:      NewCache(30 * 24 * time.Hour),
    }
}

// GetElevation retrieves elevation for a geographic point
// p: Geographic point
// Returns: Elevation in meters or error
func (o *OpenElevationProvider) GetElevation(p Point) (int, error) {
    cacheKey := fmt.Sprintf("elevation_%f_%f", p.Lat, p.Lon)
    if val, found := o.cache.Get(cacheKey); found {
        return val.(int), nil
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := o.limiter.Wait(ctx); err != nil {
        return 0, err
    }

    // Build request body
    body := fmt.Sprintf(`{"locations":[{"latitude":%f,"longitude":%f}]}`, p.Lat, p.Lon)
    resp, err := o.httpClient.Post(o.baseURL, "application/json", strings.NewReader(body))
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    // Parse response
    var result struct {
        Results []struct {
            Elevation float64 `json:"elevation"`
        } `json:"results"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, err
    }

    if len(result.Results) == 0 {
        return 0, errors.New("elevation not found")
    }

    elevation := int(math.Round(result.Results[0].Elevation))
    o.cache.Set(cacheKey, elevation)
    return elevation, nil
}

// BatchGetElevation retrieves elevations for multiple points concurrently
// points: Slice of geographic points
// Returns: Slice of elevations or first error encountered
func (o *OpenElevationProvider) BatchGetElevation(points []Point) ([]int, error) {
    type task struct {
        index int
        point Point
    }
    type result struct {
        index int
        value int
        err   error
    }

    tasks := make(chan task, len(points))
    results := make(chan result, len(points)))
    var wg sync.WaitGroup

    // Determine optimal worker count
    numWorkers := 8
    if numWorkers > len(points) {
        numWorkers = len(points)
    }

    // Start workers
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for t := range tasks {
                elev, err := o.GetElevation(t.point)
                results <- result{t.index, elev, err}
            }
        }()
    }

    // Send tasks
    go func() {
        for i, p := range points {
            tasks <- task{i, p}
        }
        close(tasks)
    }()

    // Close results channel when done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    elevations := make([]int, len(points))
    for res := range results {
        if res.err != nil {
            return nil, res.err
        }
        elevations[res.index] = res.value
    }

    return elevations, nil
}