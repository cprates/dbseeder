package logger

import (
	"fmt"
	"time"
)

type Logger interface {
	Log(fmt string, args ...any)
}

type TextLogger struct{}

func NewTextLogger() *TextLogger {
	return &TextLogger{}
}

func (TextLogger) Log(format string, args ...any) {
	fmt.Printf("[%s] "+format, append([]any{time.Now().Format(time.TimeOnly)}, args...)...)
}

type NullLogger struct{}

func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

func (NullLogger) Log(_ string, _ ...any) {}
