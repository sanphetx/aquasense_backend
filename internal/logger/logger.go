package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// InitLogger initializes the global zap logger
func InitLogger() {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder // Use readable timestamps
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder // Colored output for console

	// Use console encoder for development, can switch to JSON encoder for production
	consoleEncoder := zapcore.NewConsoleEncoder(config)
	
	// Check environment
	env := os.Getenv("ENV")
	if env == "production" {
		config.EncodeLevel = zapcore.LowercaseLevelEncoder // Standard string levels for JSON
		jsonEncoder := zapcore.NewJSONEncoder(config)
		core := zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), zap.InfoLevel)
		Log = zap.New(core, zap.AddCaller())
	} else {
		// Development mode
		core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel)
		Log = zap.New(core, zap.AddCaller())
	}

	zap.ReplaceGlobals(Log) // Replaces global zap logger
}

// Get returns the initialized global logger instance
func Get() *zap.Logger {
	if Log == nil {
		InitLogger()
	}
	return Log
}
