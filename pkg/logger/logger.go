package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/eyedeekay/github-archiver/pkg/util"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	SilentLevel
)

var levelNames = map[LogLevel]string{
	DebugLevel:  "DEBUG",
	InfoLevel:   "INFO ",
	WarnLevel:   "WARN ",
	ErrorLevel:  "ERROR",
	FatalLevel:  "FATAL",
	SilentLevel: "SILENT",
}

// Logger provides structured logging for the application
type Logger struct {
	level  LogLevel
	writer io.Writer
	logger *log.Logger
}

// New creates a new Logger
func New(level LogLevel, writer io.Writer) *Logger {
	return &Logger{
		level:  level,
		writer: writer,
		logger: log.New(writer, "", 0),
	}
}

// SetLevel changes the current log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// log formats and writes a log message if the level is sufficient
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Format with timestamp, level name, and message
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	levelStr := levelNames[level]
	message := fmt.Sprintf(format, args...)

	l.logger.Printf("[%s] %s: %s", timestamp, levelStr, message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Fatal logs a fatal message and exits the application
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FatalLevel, format, args...)
	os.Exit(1)
}

// Default logger
var defaultLogger = New(InfoLevel, os.Stdout)

// SetDefaultLevel sets the log level for the default logger
func SetDefaultLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// Debug logs to the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs to the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs to the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs to the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs to the default logger and exits
func Fatal(format string, args ...interface{}) {
	if util.FORCE_PROCESSING {
		defaultLogger.Error(format, args...)
	} else {
		defaultLogger.Fatal(format, args...)
	}

}
