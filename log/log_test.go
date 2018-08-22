package log

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	if _, err := NewLogger(); err != nil {
		t.Fatal(err)
	}
}

func TestNewNullLogger(t *testing.T) {
	l := NewDiscard()
	l.Info("test output", zap.String("key", "value"))
}
