package probe

import "context"

func withRetry(ctx context.Context, retries int, fn func() error) error {
	var err error

	for i := 0; i < retries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err = fn()
			if err == nil {
				return nil
			}
		}
	}
	return err
}
