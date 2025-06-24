package geoutil

import (
    "sync"
)

// FullLocation retrieves comprehensive geographic information for a point
// p: Geographic point
// geocoder: Geocoder implementation
// elevation: Elevation provider
// Returns: Complete location information
func FullLocation(p Point, geocoder Geocoder, elevation ElevationProvider) (Location, error) {
    loc, err := geocoder.ReverseGeocode(p)
    if err != nil {
        return Location{}, err
    }

    elev, err := elevation.GetElevation(p)
    if err != nil {
        return loc, err
    }

    loc.Elevation = elev
    // Placeholder for timezone (requires external library)
    loc.Timezone = "UTC"
    return loc, nil
}

// BatchFullLocation retrieves comprehensive geographic information concurrently
// points: Slice of geographic points
// geocoder: Geocoder implementation
// elevation: Elevation provider
// Returns: Slice of complete location information
func BatchFullLocation(points []Point, geocoder Geocoder, elevation ElevationProvider) ([]Location, error) {
    type task struct {
        index int
        point Point
    }
    type result struct {
        index int
        loc   Location
        err   error
    }

    tasks := make(chan task, len(points))
    results := make(chan result, len(points)))
    var wg sync.WaitGroup

    // Limit concurrent API requests
    maxConcurrent := 20
    sem := make(chan struct{}, maxConcurrent)

    // Start workers
    for i := 0; i < maxConcurrent; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for t := range tasks {
                sem <- struct{}{}
                loc, err := FullLocation(t.point, geocoder, elevation)
                <-sem
                results <- result{t.index, loc, err}
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

    // Close results when done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    locations := make([]Location, len(points))
    for res := range results {
        if res.err != nil {
            return nil, res.err
        }
        locations[res.index] = res.loc
    }

    return locations, nil
}