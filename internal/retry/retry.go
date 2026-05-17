// Package retry contains small retry helpers shared by provider HTTP clients.
package retry

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MaxRetries is the per-request cap on retry attempts after the first try.
const MaxRetries = 5

// RetryableStatus reports whether an HTTP response should be retried.
func RetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || (status >= 500 && status <= 599)
}

// Delay returns a Retry-After delay when present, otherwise exponential
// backoff with jitter.
func Delay(attempt int, retryAfter string, now func() time.Time) time.Duration {
	if d, ok := retryAfterDelay(retryAfter, now); ok {
		return d
	}
	return Backoff(attempt)
}

// Backoff returns 200ms, 400ms, 800ms, ... plus jitter, capped at 8s.
func Backoff(attempt int) time.Duration {
	const base = 200 * time.Millisecond
	d := base << attempt
	if d > 8*time.Second {
		d = 8 * time.Second
	}
	jit := time.Duration(rand.Int64N(int64(d/2))) - d/4
	return d + jit
}

// Sleep waits for d or until ctx is cancelled. Tests may pass sleepFn to avoid
// wall-clock sleeps.
func Sleep(ctx context.Context, sleepFn func(time.Duration), d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	if sleepFn != nil {
		sleepFn(d)
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryAfterDelay(value string, now func() time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0, true
		}
		return time.Duration(seconds) * time.Second, true
	}
	t, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	if now == nil {
		now = time.Now
	}
	d := t.Sub(now())
	if d < 0 {
		d = 0
	}
	return d, true
}
