package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// Init initializes the structured logger.
func Init(level string, isProduction bool) {
	var config zap.Config

	if isProduction {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Set log level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zap.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Build logger
	var err error
	Log, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(Log)
}

// Info logs an info message.
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Error logs an error message.
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Fatal logs a fatal message and exits.
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// Warn logs a warning message.
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Debug logs a debug message.
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Field helpers
func String(key string, val string) zap.Field {
	return zap.String(key, val)
}

func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func Bool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

func Err(err error) zap.Field {
	return zap.Error(err)
}

func Any(key string, val interface{}) zap.Field {
	return zap.Any(key, val)
}

func Duration(key string, val time.Duration) zap.Field {
	return zap.Duration(key, val)
}

func Time(key string, val time.Time) zap.Field {
	return zap.Time(key, val)
}
