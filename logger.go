package errguard

var (
	defaultLogger noopLogger
)

// Logger is the interface recognised by the guard.
type Logger interface {
	Log(v ...interface{}) error
}

type noopLogger struct{}

func (logger noopLogger) Log(v ...interface{}) error {
	return nil
}
