package logger

import (
	"os"

	"go.elastic.co/ecszap"
	"go.f0o.dev/netbench/interfaces"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	logger *zap.SugaredLogger
}

func NewLogger(fields ...interface{}) interfaces.Logger {
	var l zapcore.Level
	switch os.Getenv("LOG_LEVEL") {
	case "fatal", "FATAL":
		l = zapcore.FatalLevel
	case "panic", "PANIC":
		l = zapcore.PanicLevel
	case "dpanic", "DPANIC":
		l = zapcore.DPanicLevel
	case "error", "ERROR":
		l = zapcore.ErrorLevel
	case "warn", "WARN", "":
		l = zapcore.WarnLevel
	case "info", "INFO":
		l = zapcore.InfoLevel
	case "debug", "DEBUG":
		l = zapcore.DebugLevel
	default:
		panic("illegal LOG_LEVEL supplied")
	}
	r := logger{}
	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stderr, l)
	r.logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2)).Sugar().With(fields...)
	return &r
}

func (this *logger) Debug(msg string, fields ...interface{}) {
	this.logger.Debugf(msg, fields...)
}
func (this *logger) Info(msg string, fields ...interface{}) {
	this.logger.Infof(msg, fields...)
}
func (this *logger) Warn(msg string, fields ...interface{}) {
	this.logger.Warnf(msg, fields...)
}
func (this *logger) Error(msg string, fields ...interface{}) {
	this.logger.Errorf(msg, fields...)
}
func (this *logger) Fatal(msg string, fields ...interface{}) {
	this.logger.Fatalf(msg, fields...)
}
func (this *logger) Child(fields ...interface{}) interfaces.Logger {
	return &logger{logger: this.logger.With(fields...)}
}
