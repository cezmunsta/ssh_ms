package log

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGlobalLogger(t *testing.T) {
	l := GetLogger(defaultLevel, defaultOut)
	if l.GetLevel() != defaultLevel {
		t.Fatalf("expected: %v got: %v", defaultLevel, l.GetLevel())
	}

	l.SetLevel(logrus.DebugLevel)
	if l.GetLevel() != logrus.DebugLevel {
		t.Fatalf("expected: %v got: %v", defaultLevel, l.GetLevel())
	}

	nl := GetLogger(defaultLevel, defaultOut)
	if nl.GetLevel() != l.GetLevel() {
		t.Fatalf("expected: %v got: %v", l.GetLevel(), nl.GetLevel())
	}
}
