package geoutil

import "math"

// DistanceHaversine calculates great-circle distance using Haversine formula
// p1, p2: Geographic points
// Returns: Distance in kilometers
func DistanceHaversine(p1, p2 Point) float64 {
    const R = 6371 // Earth radius in km
    φ1 := p1.Lat * math.Pi / 180
    φ2 := p2.Lat * math.Pi / 180
    Δφ := (p2.Lat - p1.Lat) * math.Pi / 180
    Δλ := (p2.Lon - p1.Lon) * math.Pi / 180

    a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
        math.Cos(φ1)*math.Cos(φ2)*
            math.Sin(Δλ/2)*math.Sin(Δλ/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

    return R * c
}

// BatchDistanceConcurrent calculates distance matrix concurrently
// points: Slice of geographic points
// distanceFunc: Distance calculation function
// Returns: NxN distance matrix where matrix[i][j] = distance(points[i], points[j])
func BatchDistanceConcurrent(points []Point, distanceFunc func(p1, p2 Point) float64) [][]float64 {
    n := len(points)
    matrix := make([][]float64, n)
    for i := range matrix {
        matrix[i] = make([]float64, n)
    }

    type task struct {
        i, j int
    }
    tasks := make(chan task, n*(n-1)/2)
    results := make(chan struct{ i, j int; dist float64 }, cap(tasks))

    var wg sync.WaitGroup

    // Start workers
    numWorkers := 8
    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for t := range tasks {
                dist := distanceFunc(points[t.i], points[t.j])
                results <- struct {
                    i, j int
                    dist float64
                }{t.i, t.j, dist}
            }
        }()
    }

    // Generate tasks (only upper triangle)
    go func() {
        for i := 0; i < n; i++ {
            for j := i + 1; j < n; j++ {
                tasks <- task{i, j}
            }
        }
        close(tasks)
    }()

    // Close results when done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Fill symmetric matrix
    for res := range results {
        matrix[res.i][res.j] = res.dist
        matrix[res.j][res.i] = res.dist
    }

    return matrix
}