package errguard

import (
	"context"
	"testing"

	"github.com/jjeffery/errors"
	"github.com/jjeffery/kv"
)

func TestRetry(t *testing.T) {
	err1 := errors.New("testing error condition")
	if ShouldRetry(err1) {
		t.Errorf("want ShouldRetry=false, got true")
	}
	err2 := Retry(err1)
	if !ShouldRetry(err2) {
		t.Errorf("want ShouldRetry=true, got false")
	}
	if got, want := err2.Error(), "testing error condition"; got != want {
		t.Errorf("got=%q, want=%q", got, want)
	}
	if got, want := errors.Cause(err2), err1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := Retry(nil); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestGuard(t *testing.T) {
	var logs []kv.List
	var attempt int
	var guard Guard
	guard.Logger = loggerFunc(func(v ...interface{}) error {
		logs = append(logs, kv.List(v))
		return nil
	})
	guard.Run(context.Background(), func() error {
		attempt++
		if attempt < 3 {
			return Retry(errors.New("test error"))
		}
		return nil
	})

	if got, want := len(logs), 2; got != want {
		t.Errorf("got=%v, want=%v", got, want)
	}

	for i, l := range logs {
		if got, want := getInt(l, "attempt"), i+1; got != want {
			t.Errorf("attempt: got=%v, want=%v", got, want)
		}
		if i == 0 {
			if got, want := getString(l, "level"), "info"; got != want {
				t.Errorf("level: got=%v, want=%v", got, want)
			}
		} else {
			if got, want := getString(l, "level"), "warn"; got != want {
				t.Errorf("level: got=%v, want=%v", got, want)
			}
		}
	}
}

func getInt(list kv.List, key string) int {
	for i := 0; i < len(list); i += 2 {
		if list[i] == key {
			return (list[i+1]).(int)
		}
	}
	return -1
}
func getString(list kv.List, key string) string {
	for i := 0; i < len(list); i += 2 {
		if list[i] == key {
			return (list[i+1]).(string)
		}
	}
	return "<not-found>"
}
