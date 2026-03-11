package shared

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

type Logger struct {
	logger   *log.Logger
	minLevel LogLevel
}

func NewLogger(minLevel LogLevel) *Logger {
	return &Logger{
		logger:   log.New(os.Stdout, "", 0),
		minLevel: minLevel,
	}
}

func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"service":   "eventbus",
	}

	for k, v := range fields {
		entry[k] = v
	}

	jsonEntry, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	l.logger.Println(string(jsonEntry))
}

func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}

	return levels[level] >= levels[l.minLevel]
}

func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log(LevelDebug, message, fields)
}

func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log(LevelInfo, message, fields)
}

func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log(LevelWarn, message, fields)
}

func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log(LevelError, message, fields)
}
