package logging

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

type Logger struct {
	mu          sync.RWMutex
	level       Level
	serviceName string
	environment string
	formatter   Formatter
	output      *os.File
}

type Formatter interface {
	Format(entry LogEntry) ([]byte, error)
}

type JSONFormatter struct {
	TimestampFormat string
	PrettyPrint     bool
}
type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       string                 `json:"level"`
	Service     string                 `json:"service"`
	Environment string                 `json:"environment"`
	Message     string                 `json:"message"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

func (f *JSONFormatter) Format(entry LogEntry) ([]byte, error) {
	if f.TimestampFormat == "" {
		f.TimestampFormat = time.RFC3339
	}
	entry.Timestamp = time.Now().Format(f.TimestampFormat)

	if f.PrettyPrint {
		return json.MarshalIndent(entry, "", "  ")
	}
	return json.Marshal(entry)
}

func NewLogger(serviceName, environment string, level Level) *Logger {
	return &Logger{
		level:       level,
		serviceName: serviceName,
		environment: environment,
		formatter:   &JSONFormatter{},
		output:      os.Stdout,
	}
}

func (l *Logger) log(ctx context.Context, level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Level:       level.String(),
		Service:     l.serviceName,
		Environment: l.environment,
		Message:     msg,
		Fields:      fields,
	}

	// Extract trace/span IDs from context if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		entry.TraceID = traceID.(string)
	}
	if spanID := ctx.Value("span_id"); spanID != nil {
		entry.SpanID = spanID.(string)
	}

	data, err := l.formatter.Format(entry)
	if err != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.output.Write(data)
	l.output.Write([]byte("\n"))
}

func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, DEBUG, msg, fields)
}

func (l *Logger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, INFO, msg, fields)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, WARN, msg, fields)
}

func (l *Logger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, ERROR, msg, fields)
}
func (l *Logger) Fatal(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, FATAL, msg, fields)
	os.Exit(1)
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		level:       l.level,
		serviceName: l.serviceName,
		environment: l.environment,
		formatter:   l.formatter,
		output:      l.output,
	}
}

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
