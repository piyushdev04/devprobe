package probe

import (
	"context"
	"net/http"
	"sync"

	"devprobe/internal"
)

func HTTP(ctx context.Context, rawURL string, retries int, ch chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 0,
	}

	var (
		status   string
		duration int64
	)

	err := withRetry(ctx, retries, func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return err
		}

		ms, err := internal.Measure(func() error {
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			status = resp.Status
			return nil
		})
		duration = ms
		return err
	})

	ch <- Result{
		Name:     "HTTP request",
		Duration: duration,
		Err:      err,
		Extra:    status,
		Order:    4,
	}
}
