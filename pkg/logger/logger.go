package logger

import (
	"context"
	"encoding/hex"
	"fmt"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Logger is the interface used to get log output from pgx.
type Logger interface {
	// Log a message at the given level with data key/value pairs. data may be nil.
	Log(ctx context.Context, level LogLevel, msg string, data map[string]any)
}

// Func is a wrapper around a function to satisfy the pgx.Logger interface
type Func func(ctx context.Context, level LogLevel, msg string, data map[string]interface{})

// Log delegates the logging request to the wrapped function
func (f Func) Log(ctx context.Context, level LogLevel, msg string, data map[string]interface{}) {
	f(ctx, level, msg, data)
}

func NewLogger(l *logrus.Logger) Logger {
	return &logger{logger: l}
}

type logger struct {
	logger *logrus.Logger
}

func (l *logger) Log(ctx context.Context, level LogLevel, msg string, data map[string]any) {
	logger := logrus.NewEntry(l.logger)
	logger = logger.WithContext(ctx)
	if data != nil {
		logger = logger.WithFields(data)
	}

	switch level {
	case LogLevelTrace:
		logger.WithField("PGX_LOG_LEVEL", level).Debug(msg)
	case LogLevelDebug:
		logger.Debug(msg)
	case LogLevelInfo:
		logger.Info(msg)
	case LogLevelWarn:
		logger.Warn(msg)
	case LogLevelError:
		logger.Error(msg)
	default:
		logger.WithField("INVALID_PGX_LOG_LEVEL", level).Error(msg)
	}
}

// LogLevel represents the pgx logging level. See LogLevel* constants for
// possible values.
type LogLevel int

// The values for log levels are chosen such that the zero value means that no
// log level was specified.
const (
	LogLevelTrace = LogLevel(6)
	LogLevelDebug = LogLevel(5)
	LogLevelInfo  = LogLevel(4)
	LogLevelWarn  = LogLevel(3)
	LogLevelError = LogLevel(2)
	LogLevelNone  = LogLevel(1)
)

func (ll LogLevel) String() string {
	switch ll {
	case LogLevelTrace:
		return "trace"
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelNone:
		return "none"
	default:
		return fmt.Sprintf("invalid level %d", ll)
	}
}

// LogLevelFromString converts log level string to constant
//
// Valid levels:
//
//	trace
//	debug
//	info
//	warn
//	error
//	none
func LogLevelFromString(s string) (LogLevel, error) {
	switch s {
	case "trace":
		return LogLevelTrace, nil
	case "debug":
		return LogLevelDebug, nil
	case "info":
		return LogLevelInfo, nil
	case "warn":
		return LogLevelWarn, nil
	case "error":
		return LogLevelError, nil
	case "none":
		return LogLevelNone, nil
	default:
		return 0, errors.New("invalid log level")
	}
}

func LogQueryArgs(args []any) []any {
	logArgs := make([]any, 0, len(args))

	for _, a := range args {
		switch v := a.(type) {
		case []byte:
			if len(v) < 64 {
				a = hex.EncodeToString(v)
			} else {
				a = fmt.Sprintf("%x (truncated %d bytes)", v[:64], len(v)-64)
			}
		case string:
			if len(v) > 64 {
				var l int
				for w := 0; l < 64; l += w {
					_, w = utf8.DecodeRuneInString(v[l:])
				}
				if len(v) > l {
					a = fmt.Sprintf("%s (truncated %d bytes)", v[:l], len(v)-l)
				}
			}
		}
		logArgs = append(logArgs, a)
	}

	return logArgs
}
