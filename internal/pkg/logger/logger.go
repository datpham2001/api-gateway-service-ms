package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Standard field names
const (
	FieldRequestID  = "request_id"
	FieldUserID     = "user_id"
	FieldMethod     = "method"
	FieldPath       = "path"
	FieldStatusCode = "status_code"
	FieldError      = "error"
	FieldDuration   = "duration"
	FieldIP         = "ip"
	FieldService    = "service"
	FieldComponent  = "component"
	FieldFile       = "file"
	FieldLine       = "line"
	FieldFunc       = "func"
)

var (
	// defaultLogger is the default logger instance
	defaultLogger *Logger
	once          sync.Once
)

// Add this new CustomFormatter struct
type CustomFormatter struct {
	TimestampFormat string
	ShowColors      bool
}

// Add the Format method for the CustomFormatter
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Colors for different log levels
	var levelColor string
	if f.ShowColors {
		switch entry.Level {
		case logrus.DebugLevel:
			levelColor = "\033[37m" // White
		case logrus.InfoLevel:
			levelColor = "\033[32m" // Green
		case logrus.WarnLevel:
			levelColor = "\033[33m" // Yellow
		case logrus.ErrorLevel:
			levelColor = "\033[31m" // Red
		case logrus.FatalLevel:
			levelColor = "\033[35m" // Purple
		default:
			levelColor = "\033[37m" // Default white
		}
	}

	// Format timestamp
	timestamp := entry.Time.Format(f.TimestampFormat)

	// Format level
	level := strings.ToUpper(entry.Level.String())
	if f.ShowColors {
		level = fmt.Sprintf("%s%-6s\033[0m", levelColor, level)
	}

	// Format message
	message := entry.Message

	// Format caller info if available
	var caller string
	if entry.HasCaller() {
		caller = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
	}

	// Format fields
	var fields string
	if len(entry.Data) > 0 {
		var fieldStrings []string
		for k, v := range entry.Data {
			// Skip caller fields as we handle them separately
			if k != "file" && k != "line" && k != "func" {
				fieldStrings = append(fieldStrings, fmt.Sprintf("%s=%v", k, v))
			}
		}
		if len(fieldStrings) > 0 {
			if f.ShowColors {
				fields = "\033[36m[" + strings.Join(fieldStrings, "] [") + "]\033[0m"
			} else {
				fields = "[" + strings.Join(fieldStrings, "] [") + "]"
			}
		}
	}

	// Build the log line
	var logLine string
	if caller != "" {
		if f.ShowColors {
			logLine = fmt.Sprintf("%s | %s | %s %s \033[90m(%s)\033[0m\n",
				timestamp, level, message, fields, caller)
		} else {
			logLine = fmt.Sprintf("%s | %s | %s %s (%s)\n",
				timestamp, level, message, fields, caller)
		}
	} else {
		logLine = fmt.Sprintf("%s | %s | %s %s\n",
			timestamp, level, message, fields)
	}

	return []byte(logLine), nil
}

// Config holds the logger configuration
type LoggerConfig struct {
	// Environment sets the environment (development, staging, production)
	Env string
	// Level sets the logging level
	Level logrus.Level
	// ServiceName is the name of the service
	ServiceName string
	// Output where logs are written to
	Output io.Writer
	// EnableCaller includes caller information (file, line)
	EnableCaller bool
	// CallerSkip frames to skip for caller
	CallerSkip int
	// EnableJSON enables JSON formatting (auto-enabled in production)
	EnableJSON bool
	// Fields are additional fields to include with every log entry
	Fields map[string]interface{}
}

// Logger wraps logrus.Logger
type Logger struct {
	*logrus.Logger
	config    LoggerConfig
	baseEntry *logrus.Entry
}

// New creates a new configured logger
func New(config LoggerConfig) *Logger {
	if config.Output == nil {
		config.Output = os.Stdout
	}

	logger := logrus.New()
	logger.SetOutput(config.Output)
	logger.SetLevel(config.Level)

	// In production or when explicitly enabled, use JSON formatter
	if config.Env == "production" || config.EnableJSON {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:   time.RFC3339Nano,
			DisableHTMLEscape: true,
		})
	} else {
		// In development, use a more readable format
		logger.SetFormatter(&CustomFormatter{
			TimestampFormat: time.RFC3339Nano,
			ShowColors:      true,
		})
	}

	// Create base entry with service information
	baseFields := logrus.Fields{
		FieldService: config.ServiceName,
	}

	// Add any custom fields
	for k, v := range config.Fields {
		baseFields[k] = v
	}

	l := &Logger{
		Logger:    logger,
		config:    config,
		baseEntry: logger.WithFields(baseFields),
	}

	return l
}

// SetupLogger initializes the default logger
func SetupLogger(config LoggerConfig) *Logger {
	once.Do(func() {
		defaultLogger = New(config)
	})

	return defaultLogger
}

// WithRequestID adds a request ID to the logger entry
func (l *Logger) WithRequestID(requestID string) *logrus.Entry {
	return l.baseEntry.WithField(FieldRequestID, requestID)
}

// WithUserID adds a user ID to the logger entry
func (l *Logger) WithUserID(userID string) *logrus.Entry {
	return l.baseEntry.WithField(FieldUserID, userID)
}

// WithContext adds context-specific fields to the logger entry
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.baseEntry

	// Extract values from context if they exist
	// This is where you'd pull request IDs, correlation IDs, etc. from your context
	if ctx != nil {
		if requestID, ok := ctx.Value("request_id").(string); ok && requestID != "" {
			entry = entry.WithField(FieldRequestID, requestID)
		}
		if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
			entry = entry.WithField(FieldUserID, userID)
		}
	}

	return entry
}

// WithError adds an error to the logger entry
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.baseEntry.WithError(err)
}

// WithComponent adds component information to the logger entry
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.baseEntry.WithField(FieldComponent, component)
}

// WithFields adds multiple fields to the logger entry
func (l *Logger) WithFields(fields map[string]interface{}) *logrus.Entry {
	return l.baseEntry.WithFields(logrus.Fields(fields))
}

// Field creates a logging field for a map of fields
func Field(key string, value interface{}) map[string]interface{} {
	return map[string]interface{}{key: value}
}

// addCallerInfo adds caller information to the entry if enabled
func (l *Logger) addCallerInfo(entry *logrus.Entry) *logrus.Entry {
	if !l.config.EnableCaller {
		return entry
	}

	// Skip 2 frames by default (this function + the calling function)
	// Add CallerSkip for additional frame skipping
	skip := 2 + l.config.CallerSkip

	if pc, file, line, ok := runtime.Caller(skip); ok {
		entry = entry.WithFields(logrus.Fields{
			FieldFile: trimFilePath(file),
			FieldLine: line,
			FieldFunc: runtime.FuncForPC(pc).Name(),
		})
	}

	return entry
}

// trimFilePath trims the path from file names for cleaner logs
func trimFilePath(path string) string {
	// Find the last instance of '/src/' in the path and trim everything before it
	i := strings.LastIndex(path, "/src/")
	if i > 0 {
		return path[i+5:]
	}
	// If not found, just return the base name
	return path[strings.LastIndex(path, "/")+1:]
}

// Debug logs a debug message with optional fields
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	entry := l.baseEntry
	for _, f := range fields {
		entry = entry.WithFields(logrus.Fields(f))
	}
	l.addCallerInfo(entry).Debug(msg)
}

// Info logs an info message with optional fields
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	entry := l.baseEntry
	for _, f := range fields {
		entry = entry.WithFields(logrus.Fields(f))
	}
	l.addCallerInfo(entry).Info(msg)
}

// Warn logs a warning message with optional fields
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	entry := l.baseEntry
	for _, f := range fields {
		entry = entry.WithFields(logrus.Fields(f))
	}
	l.addCallerInfo(entry).Warn(msg)
}

// Error logs an error message with optional fields
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	entry := l.baseEntry
	for _, f := range fields {
		entry = entry.WithFields(logrus.Fields(f))
	}
	l.addCallerInfo(entry).Error(msg)
}

// Fatal logs a fatal message with optional fields and exits
func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
	entry := l.baseEntry
	for _, f := range fields {
		entry = entry.WithFields(logrus.Fields(f))
	}
	l.addCallerInfo(entry).Fatal(msg)
}

// Debugf logs a debug message with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.addCallerInfo(l.baseEntry).Debugf(format, args...)
}

// Infof logs an info message with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.addCallerInfo(l.baseEntry).Infof(format, args...)
}

// Warnf logs a warning message with formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.addCallerInfo(l.baseEntry).Warnf(format, args...)
}

// Errorf logs an error message with formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.addCallerInfo(l.baseEntry).Errorf(format, args...)
}

// Fatalf logs a fatal message with formatting and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.addCallerInfo(l.baseEntry).Fatalf(format, args...)
}

// ErrorErr logs an error with a message
func (l *Logger) ErrorErr(err error, msg string, fields ...map[string]interface{}) {
	if err == nil {
		return
	}
	entry := l.baseEntry.WithError(err)
	for _, f := range fields {
		entry = entry.WithFields(logrus.Fields(f))
	}
	l.addCallerInfo(entry).Error(msg)
}

// Debug uses the default logger to log a debug message
func Debug(msg string, fields ...map[string]interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, fields...)
	}
}

// Info uses the default logger to log an info message
func Info(msg string, fields ...map[string]interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, fields...)
	}
}

// Warn uses the default logger to log a warning message
func Warn(msg string, fields ...map[string]interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, fields...)
	}
}

// Error uses the default logger to log an error message
func Error(msg string, fields ...map[string]interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, fields...)
	}
}

// Fatal uses the default logger to log a fatal message and exit
func Fatal(msg string, fields ...map[string]interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatal(msg, fields...)
	}
}

// Debugf uses the default logger to log a debug message with formatting
func Debugf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debugf(format, args...)
	}
}

// Infof uses the default logger to log an info message with formatting
func Infof(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Infof(format, args...)
	}
}

// Warnf uses the default logger to log a warning message with formatting
func Warnf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warnf(format, args...)
	}
}

// Errorf uses the default logger to log an error message with formatting
func Errorf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Errorf(format, args...)
	}
}

// Fatalf uses the default logger to log a fatal message with formatting and exit
func Fatalf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatalf(format, args...)
	}
}

// ErrorErr uses the default logger to log an error with a message
func ErrorErr(err error, msg string, fields ...map[string]interface{}) {
	if defaultLogger != nil {
		defaultLogger.ErrorErr(err, msg, fields...)
	}
}

// WithRequestID adds a request ID to the default logger entry
func WithRequestID(requestID string) *logrus.Entry {
	if defaultLogger != nil {
		return defaultLogger.WithRequestID(requestID)
	}
	return logrus.WithField(FieldRequestID, requestID)
}

// WithContext adds context-specific fields to the default logger entry
func WithContext(ctx context.Context) *logrus.Entry {
	if defaultLogger != nil {
		return defaultLogger.WithContext(ctx)
	}
	return logrus.NewEntry(logrus.StandardLogger())
}

// WithError adds an error to the default logger entry
func WithError(err error) *logrus.Entry {
	if defaultLogger != nil {
		return defaultLogger.WithError(err)
	}
	return logrus.WithError(err)
}

// WithComponent adds component information to the default logger entry
func WithComponent(component string) *logrus.Entry {
	if defaultLogger != nil {
		return defaultLogger.WithComponent(component)
	}
	return logrus.WithField(FieldComponent, component)
}

// WithFields adds multiple fields to the default logger entry
func WithFields(fields map[string]interface{}) *logrus.Entry {
	if defaultLogger != nil {
		return defaultLogger.WithFields(fields)
	}
	return logrus.WithFields(logrus.Fields(fields))
}

func SetConfig(env string) {
	defaultLogger.config.Env = env
}

func SetLevel(level logrus.Level) {
	defaultLogger.SetLevel(level)
}
