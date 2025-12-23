package probe

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type LoadStats struct {
	Total     int
	Success   int
	Errors    int
	Latencies []int64
}

func Load(ctx context.Context, url string, concurrency, total, retries int) LoadStats {
	client := &http.Client{}
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup
	stats := LoadStats{
		Total:     total,
		Latencies: make([]int64, 0, total),
	}

	var mu sync.Mutex

	for i := 0; i < total; i++ {
		select {
		case <-ctx.Done():
			return stats
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			var elapsed int64
			err := withRetry(ctx, retries, func() error {
				start := time.Now()

				req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
				if err != nil {
					return err
				}

				resp, err := client.Do(req)
				elapsed = time.Since(start).Milliseconds()

				if err != nil {
					return err
				}

				return resp.Body.Close()
			})

			mu.Lock()
			defer mu.Unlock()

			stats.Latencies = append(stats.Latencies, elapsed)

			if err != nil {
				stats.Errors++
			} else {
				stats.Success++
			}
		}()
	}

	wg.Wait()
	return stats
}
