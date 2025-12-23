package probe

import (
	"context"
	"net"
	"net/url"
	"sync"

	"devprobe/internal"
)

func TCP(ctx context.Context, rawURL string, retries int, ch chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	u, _ := url.Parse(rawURL)
	host := u.Host
	if u.Port() == "" {
		if u.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	var duration int64
	err := withRetry(ctx, retries, func() error {
		dialer := net.Dialer{}
		ms, err := internal.Measure(func() error {
			conn, err := dialer.DialContext(ctx, "tcp", host)
			if err != nil {
				return err
			}
			return conn.Close()
		})
		duration = ms
		return err
	})

	ch <- Result{
		Name:     "TCP connect",
		Duration: duration,
		Err:      err,
		Order:    2,
	}
}
