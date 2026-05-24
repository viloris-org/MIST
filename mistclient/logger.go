package mistclient

// Logger is the logging interface used by the client library.
// Platforms can inject their own implementation (e.g. logrus for CLI,
// android.util.Log for Android, os_log for iOS).
type Logger interface {
	Info(args ...any)
	Infof(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Debug(args ...any)
	Debugf(format string, args ...any)
	Warn(args ...any)
	Warnf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

type nopLogger struct{}

func (nopLogger) Info(args ...any)                  {}
func (nopLogger) Infof(format string, args ...any)  {}
func (nopLogger) Error(args ...any)                 {}
func (nopLogger) Errorf(format string, args ...any) {}
func (nopLogger) Debug(args ...any)                 {}
func (nopLogger) Debugf(format string, args ...any) {}
func (nopLogger) Warn(args ...any)                  {}
func (nopLogger) Warnf(format string, args ...any)  {}
func (nopLogger) Fatal(args ...any)                 {}
func (nopLogger) Fatalf(format string, args ...any) {}
