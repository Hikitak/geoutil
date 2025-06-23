package geoutil

// IsPointInPolygon проверяет принадлежность точки полигону
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

// FilterPointsInPolygonConcurrent конкурентная фильтрация точек
func FilterPointsInPolygonConcurrent(points []Point, polygon []Point) []Point {
	type result struct {
		index int
		valid bool
	}

	results := make(chan result, len(points))
	var wg sync.WaitGroup

	// Определяем размер батча
	batchSize := 1000
	if len(points) < 1000 {
		batchSize = len(points)
	}

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

	go func() {
		wg.Wait()
		close(results)
	}()

	filtered := make([]Point, 0, len(points))
	for res := range results {
		if res.valid {
			filtered = append(filtered, points[res.index])
		}
	}

	return filtered
}