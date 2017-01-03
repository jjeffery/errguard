package errguard

import (
	"context"
	"errors"
	"log"
	"math/rand"
)

func Example() {
	ctx := context.TODO()
	var guard Guard
	guard.Logger = loggerFunc(func(v ...interface{}) error {
		log.Println(v...)
		return nil
	})

	guard.Run(ctx, func() error {
		return doSomethingWith(ctx)
	})
}

func doSomethingWith(ctx context.Context) error {
	log.Println("doing something")
	// has a 90% chance of failure
	if rand.Intn(10) > 0 {
		// Retry will mark this error as being retryable
		return Retry(errors.New("error condition"))
	}
	return nil
}

type loggerFunc func(v ...interface{}) error

func (f loggerFunc) Log(v ...interface{}) error {
	return f(v...)
}
