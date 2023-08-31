package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger returns a new instance of a ClusterIQ logger
func NewLogger() *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	loggerConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: true,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stdout",
		},
	}

	logLevel := strings.ToLower(os.Getenv("CIQ_LOG_LEVEL"))
	if logLevel == "debug" {
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		loggerConfig.DisableStacktrace = false
		loggerConfig.DisableCaller = false
	}

	logger := zap.Must(loggerConfig.Build())

	return logger
}
