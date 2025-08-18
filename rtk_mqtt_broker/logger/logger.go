package logger

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

func New(level string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[RTK-MQTT] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Info(msg string) {
	l.Printf("[INFO] %s", msg)
}

func (l *Logger) Error(msg string) {
	l.Printf("[ERROR] %s", msg)
}

func (l *Logger) Warn(msg string) {
	l.Printf("[WARN] %s", msg)
}

func (l *Logger) Debug(msg string) {
	l.Printf("[DEBUG] %s", msg)
}