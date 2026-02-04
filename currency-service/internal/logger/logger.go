package logger

import (
	"go.uber.org/zap"
)

// Log is the global logger instance
var Log *zap.Logger

// Init initializes the global logger based on environment
func Init(env string) error {
	var err error

	if env == "production" {
		// Production: JSON format, Info level and above
		Log, err = zap.NewProduction()
	} else {
		// Development: console format with colors, Debug level and above
		Log, err = zap.NewDevelopment()
	}

	if err != nil {
		return err
	}

	// Replace global zap logger
	zap.ReplaceGlobals(Log)

	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
