package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	logger *zap.SugaredLogger
}

var DefaultLogger logger

func init() {
	var z zap.Config
	atom := zap.NewAtomicLevel()
	switch os.Getenv("LOG_LEVEL") {
	case "fatal", "FATAL":
		atom.SetLevel(zapcore.FatalLevel)
	case "panic", "PANIC":
		atom.SetLevel(zapcore.PanicLevel)
	case "dpanic", "DPANIC":
		atom.SetLevel(zapcore.DPanicLevel)
	case "error", "ERROR":
		atom.SetLevel(zapcore.ErrorLevel)
	case "warn", "WARN", "":
		atom.SetLevel(zapcore.WarnLevel)
	case "info", "INFO":
		atom.SetLevel(zapcore.InfoLevel)
	case "debug", "DEBUG":
		atom.SetLevel(zapcore.DebugLevel)
	default:
		panic("illegal LOG_LEVEL supplied")
	}

	if os.Getenv("ENV") == "prod" {
		z = zap.NewProductionConfig()
	} else {
		z = zap.NewDevelopmentConfig()
	}

	z.Level = atom
	t, _ := z.Build(zap.AddCallerSkip(1))
	DefaultLogger = logger{
		logger: t.Sugar(),
	}
}

// printf styled loggers
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
