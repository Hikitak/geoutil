package geoutil

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// FullLocation возвращает полную информацию о точке
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
	// Для временных зон рекомендуется использовать внешнюю библиотеку
	loc.Timezone = "UTC" // Заглушка
	return loc, nil
}

// BatchFullLocation конкурентное получение полной информации
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

	// Ограничиваем количество одновременных запросов
	maxConcurrent := runtime.NumCPU() * 2
	if maxConcurrent > 20 {
		maxConcurrent = 20
	}
	sem := make(chan struct{}, maxConcurrent)

	// Запускаем воркеры
	for i := 0; i < runtime.NumCPU(); i++ {
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

	// Отправляем задачи
	go func() {
		for i, p := range points {
			tasks <- task{i, p}
		}
		close(tasks)
	}()

	// Закрываем канал результатов после завершения
	go func() {
		wg.Wait()
		close(results)
	}()

	locations := make([]Location, len(points))
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		locations[res.index] = res.loc
	}

	return locations, nil
}