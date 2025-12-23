package internal

import "time"

func Measure(fn func() error) (int64, error) {
	start := time.Now()
	err := fn()
	return time.Since(start).Milliseconds(), err
}