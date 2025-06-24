package geoutil

// IsPointInPolygon determines if a point is inside a polygon using ray casting algorithm
// p: Point to check
// polygon: Vertices of polygon (must have at least 3 points)
// Returns: true if point is inside polygon, false otherwise
func IsPointInPolygon(p Point, polygon []Point) bool {
    if len(polygon) < 3 {
        return false
    }

    inside := false
    j := len(polygon) - 1
    for i := 0; i < len(polygon); i++ {
        if (polygon[i].Lon > p.Lon) != (polygon[j].Lon > p.Lon) &&
            p.Lat < (polygon[j].Lat-polygon[i].Lat)*(p.Lon-polygon[i].Lon)/
                (polygon[j].Lon-polygon[i].Lon)+polygon[i].Lat {
            inside = !inside
        }
        j = i
    }
    return inside
}

// FilterPointsInPolygonConcurrent filters points inside polygon concurrently
// points: Slice of points to filter
// polygon: Polygon vertices
// Returns: Points located inside the polygon
func FilterPointsInPolygonConcurrent(points []Point, polygon []Point) []Point {
    type result struct {
        index int
        valid bool
    }

    results := make(chan result, len(points))
    var wg sync.WaitGroup

    // Determine batch size
    batchSize := 1000
    if len(points) < 1000 {
        batchSize = len(points)
    }

    // Process points in batches
    for i := 0; i < len(points); i += batchSize {
        end := i + batchSize
        if end > len(points) {
            end = len(points)
        }

        wg.Add(1)
        go func(start, end int) {
            defer wg.Done()
            for j := start; j < end; j++ {
                valid := IsPointInPolygon(points[j], polygon)
                results <- result{j, valid}
            }
        }(i, end)
    }

    // Close results when done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect valid points
    filtered := make([]Point, 0, len(points))
    for res := range results {
        if res.valid {
            filtered = append(filtered, points[res.index])
        }
    }

    return filtered
}