package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(os.Stdout)
}

// SetStandardFields sets standard fields that should be included in all log entries
func SetStandardFields(serviceName, version string) {
	logger = logger.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
	}).Logger
}

// SetLevel sets the logging level
func SetLevel(level logrus.Level) {
	logger.SetLevel(level)
}

// WithField returns a logger with the specified field
func WithField(key string, value interface{}) *logrus.Entry {
	return logger.WithField(key, value)
}

// WithFields returns a logger with the specified fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return logger.WithFields(fields)
}

// WithError returns a logger with the error field set
func WithError(err error) *logrus.Entry {
	return logger.WithError(err)
}

// Debug logs a debug message
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Info logs an info message
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Debugf logs a debug message with formatting
func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// Infof logs an info message with formatting
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Warnf logs a warning message with formatting
func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

// Errorf logs an error message with formatting
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

// Fatalf logs a fatal message with formatting and exits
func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}
