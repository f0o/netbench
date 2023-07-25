package logger

import "go.f0o.dev/netbench/interfaces"

var DefaultLogger interfaces.Logger = NewLogger()

func Debug(msg string, fields ...interface{}) {
	DefaultLogger.Debug(msg, fields...)
}
func Info(msg string, fields ...interface{}) {
	DefaultLogger.Info(msg, fields...)
}
func Warn(msg string, fields ...interface{}) {
	DefaultLogger.Warn(msg, fields...)
}
func Error(msg string, fields ...interface{}) {
	DefaultLogger.Error(msg, fields...)
}
func Fatal(msg string, fields ...interface{}) {
	DefaultLogger.Fatal(msg, fields...)
}
func Child(fields ...interface{}) interfaces.Logger {
	return DefaultLogger.Child(fields...)
}
