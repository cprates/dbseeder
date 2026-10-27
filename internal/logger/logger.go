package logger

import "fmt"

type Logger interface {
	Log(fmt string, args ...any)
}

type TextLogger struct{}

func NewTextLogger() *TextLogger {
	return &TextLogger{}
}

func (TextLogger) Log(format string, args ...any) {
	fmt.Printf(format, args...)
}

type NullLogger struct{}

func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

func (NullLogger) Log(_ string, _ ...any) {}
