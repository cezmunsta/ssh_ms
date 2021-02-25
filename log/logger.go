package log

import (
	"log/syslog"
	"os"

	"github.com/sirupsen/logrus"
	syslogHook "github.com/sirupsen/logrus/hooks/syslog"
)

var (
	logger *logrus.Logger
)

func init() {
	logger, _ = GetLoggers(logrus.DebugLevel, "")
}

// Println out to console
func Println(args ...interface{}) {
	logrus.Println(args...)
}

// Panic logs a message and panics
func Panic(args ...interface{}) {
	logrus.Panic(args...)
}

// Fatal error messages
func Fatal(args ...interface{}) {
	logrus.Fatal(args...)
}

// Error messages
func Error(args ...interface{}) {
	logrus.Error(args...)
}

// Warning messages
func Warning(args ...interface{}) {
	logrus.Warning(args...)
}

// Info messages
func Info(args ...interface{}) {
	logrus.Info(args...)
}

// Debug messages
func Debug(args ...interface{}) {
	logrus.Debug(args...)
}

// GetLoggers returns the pre-configured loggers
func GetLoggers(level logrus.Level, logFile string) (*logrus.Logger, error) {
	logrus.SetLevel(level)

	userLogger := logrus.New()
	userLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors:          true,
		DisableLevelTruncation: true,
		FullTimestamp:          true,
	})
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		defer file.Close()

		if err == nil {
			userLogger.SetOutput(file)
		} else {
			userLogger.Warn("Failed to configure logging to file")
		}
	}

	sysLogger, err := syslogHook.NewSyslogHook("", "", syslog.LOG_WARNING, "")
	if err == nil {
		userLogger.Hooks.Add(sysLogger)
	}
	return userLogger, nil
}
