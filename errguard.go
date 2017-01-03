// Package errguard makes it easy to retry error conditions that have a
// resonable chance of succeeding after a short pause. These error conditions
// include optimistic locking and deadlock.
package errguard

import (
	"context"
	"time"

	"github.com/go-stack/stack"
	"github.com/jjeffery/kv"
)

// Guard is used to retry error conditions that have a reasonable
// chance of succeeding.
type Guard struct {
	// ShouldRetry returns true if the guard should retry
	// after encountering error err. If nil, the default
	// logic is used.
	ShouldRetry func(err error) bool

	// If logger is set, the guard will log a message every
	// time the guard encounters and error and retries.
	Logger Logger
}

// Retry wraps err to return an error that indicates
// the operation should be retried by the guard.
// If err is nil returns nil.
func Retry(err error) error {
	if err == nil {
		return nil
	}
	return retryT{cause: err}
}

type retryT struct {
	cause error
}

func (err retryT) Error() string {
	return err.cause.Error()
}

func (err retryT) Cause() error {
	return err.cause
}

func (err retryT) ShouldRetry() bool {
	return true
}

var (
	// ShouldRetry is the default test for whether
	// a guard should retry after encountering error err.
	// This value is used if not specified for an individual
	// guard.
	ShouldRetry func(err error) bool

	// DefaultLogger is the default logger to use if
	// not specified for an individual guard.
	DefaultLogger Logger
)

func init() {
	ShouldRetry = func(err error) bool {
		type shouldRetryer interface {
			ShouldRetry() bool
		}
		if shouldRetry, ok := err.(shouldRetryer); ok {
			return shouldRetry.ShouldRetry()
		}
		return false
	}

	DefaultLogger = noopLogger{}
}

// Run function f and keep retrying while it returns
// a retryable error.
func (g *Guard) Run(ctx context.Context, f func() error) error {
	shouldRetry := g.ShouldRetry
	if shouldRetry == nil {
		shouldRetry = ShouldRetry
	}

	logger := g.Logger
	if logger == nil {
		logger = DefaultLogger
	}

	sleepDuration := time.Millisecond * 100
	for attempt := 1; ; attempt++ {
		err := f()
		if err == nil {
			return nil
		}
		if !shouldRetry(err) {
			return err
		}

		// At this point an optimistic locking exception has occurred.
		// Log a message and if there is still time, wait and retry.
		var level string
		if attempt <= 1 {
			level = "info"
		} else {
			level = "warn"
		}

		keyvals := []interface{}{
			kv.P("level", level),
			err,
			kv.P("caller", stack.Caller(1)),
			kv.P("attempt", attempt),
		}
		logger.Log(kv.Flatten(keyvals)...)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
			sleepDuration = sleepDuration * 2
		}
	}
}
