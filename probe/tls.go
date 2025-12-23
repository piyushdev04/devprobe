package probe

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"sync"

	"devprobe/internal"
)

func TLS(ctx context.Context, rawURL string, retries int, ch chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	u, _ := url.Parse(rawURL)
	if u.Scheme != "https" {
		ch <- Result{
			Name:  "TLS handshake",
			Extra: "skipped (http)",
			Order: 3,
		}
		return
	}

	addr := u.Host
	if u.Port() == "" {
		addr += ":443"
	}

	var duration int64
	err := withRetry(ctx, retries, func() error {
		ms, err := internal.Measure(func() error {
			dialer := &net.Dialer{}
			conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
				ServerName: u.Hostname(),
			})
			if err != nil {
				return err
			}
			return conn.Close()
		})
		duration = ms
		return err
	})

	ch <- Result{
		Name:     "TLS handshake",
		Duration: duration,
		Err:      err,
		Order:    3,
	}
}
