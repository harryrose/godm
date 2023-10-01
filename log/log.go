package log

import (
	"fmt"
	"github.com/harryrose/godm/log/levels"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

func Init(level levels.Level) error {
	cfg := zap.NewProductionConfig()

	zapLevel := zapcore.InfoLevel
	switch level {
	case levels.Debug:
		zapLevel = zapcore.DebugLevel

	case levels.Info:
		zapLevel = zapcore.InfoLevel

	case levels.Warn:
		zapLevel = zapcore.WarnLevel

	case levels.Error:
		zapLevel = zapcore.ErrorLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	built, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}
	log = built.Sugar()
	return nil
}

func Errorw(msg string, keyValuePairs ...any) {
	log.Errorw(msg, keyValuePairs...)
}

func Warnw(msg string, keyValuePairs ...any) {
	log.Warnw(msg, keyValuePairs...)
}

func Infow(msg string, keyValuePairs ...any) {
	log.Infow(msg, keyValuePairs...)
}

func Debugw(msg string, keyValuePairs ...any) {
	log.Debugw(msg, keyValuePairs...)
}
