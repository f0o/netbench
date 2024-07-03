package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	logger *zap.SugaredLogger
}

var (
	DefaultLogger logger
	TraceEnabled  bool = false
)

func init() {
	var z zap.Config
	atom := zap.NewAtomicLevel()
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "fatal":
		atom.SetLevel(zapcore.FatalLevel)
	case "panic":
		atom.SetLevel(zapcore.PanicLevel)
	case "dpanic":
		atom.SetLevel(zapcore.DPanicLevel)
	case "error":
		atom.SetLevel(zapcore.ErrorLevel)
	case "warn", "":
		atom.SetLevel(zapcore.WarnLevel)
	case "info":
		atom.SetLevel(zapcore.InfoLevel)
	case "debug":
		atom.SetLevel(zapcore.DebugLevel)
	case "trace":
		atom.SetLevel(zapcore.DebugLevel)
		TraceEnabled = true
	default:
		panic("illegal LOG_LEVEL supplied")
	}

	format := strings.ToLower(os.Getenv("LOG_FORMAT"))

	if format == "json" {
		z = zap.NewProductionConfig()
	} else {
		z = zap.NewDevelopmentConfig()
	}

	z.Level = atom
	t, _ := z.Build(zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	DefaultLogger = logger{
		logger: t.Sugar(),
	}
}

// printf styled loggers
func Trace(msg string, fields ...interface{}) {
	if TraceEnabled {
		DefaultLogger.logger.Debugf(msg, fields...)
	}
}

func Debug(msg string, fields ...interface{}) {
	DefaultLogger.logger.Debugf(msg, fields...)
}

func Info(msg string, fields ...interface{}) {
	DefaultLogger.logger.Infof(msg, fields...)
}

func Warn(msg string, fields ...interface{}) {
	DefaultLogger.logger.Warnf(msg, fields...)
}

func Error(msg string, fields ...interface{}) {
	DefaultLogger.logger.Errorf(msg, fields...)
}

func Panic(msg string, fields ...interface{}) {
	DefaultLogger.logger.Panicf(msg, fields...)
}

func DPanic(msg string, fields ...interface{}) {
	DefaultLogger.logger.DPanicf(msg, fields...)
}

func Fatal(msg string, fields ...interface{}) {
	DefaultLogger.logger.Fatalf(msg, fields...)
}

// structured loggers
func Tracew(msg string, keysAndValues ...interface{}) {
	if TraceEnabled {
		DefaultLogger.logger.Debugw(msg, keysAndValues...)
	}
}

func Debugw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.logger.Debugw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	DefaultLogger.logger.Infow(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.logger.Warnw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.logger.Errorw(msg, keysAndValues...)
}

func Panicw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.logger.Panicw(msg, keysAndValues...)
}

func DPanicw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.logger.DPanicw(msg, keysAndValues...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.logger.Fatalw(msg, keysAndValues...)
}
