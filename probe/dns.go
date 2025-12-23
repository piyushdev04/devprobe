package probe

import (
	"context"
	"net"
	"net/url"
	"sync"

	"devprobe/internal"
)

func DNS(ctx context.Context, rawURL string, retries int, ch chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	u, _ := url.Parse(rawURL)
	host := u.Hostname()

	var duration int64
	err := withRetry(ctx, retries, func() error {
		ms, err := internal.Measure(func() error {
			_, err := net.DefaultResolver.LookupHost(ctx, host)
			return err
		})
		duration = ms
		return err
	})

	ch <- Result{
		Name:     "DNS lookup",
		Duration: duration,
		Err:      err,
		Order:    1,
	}
}
