package log

import (
	"log/syslog"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	syslogHook "github.com/sirupsen/logrus/hooks/syslog"
)

var (
	defaultOut   = ""
	defaultLevel = logrus.WarnLevel
	logger       *logrus.Logger
	once         sync.Once
)

func init() {
	logger = GetLogger(defaultLevel, defaultOut)
}

// Println out to console
func Println(args ...interface{}) {
	logger.Println(args...)
}

// Printf out to the console
func Printf(format string, args ...interface{}) {
	logger.Printf(format, args...)
}

// Panic logs a message and panics
func Panic(args ...interface{}) {
	logger.Panic(args...)
}

// Panicf logs a message and panics
func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}

// Fatal error messages
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Fatalf error messages
func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}

// Error messages
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Errorf messages
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

// Warning messages
func Warning(args ...interface{}) {
	logger.Warning(args...)
}

// Warningf messages
func Warningf(format string, args ...interface{}) {
	logger.Warningf(format, args...)
}

// Info messages
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Infof messages
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Debug messages
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Debugf messages
func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// SetLevel of logging
func SetLevel(level logrus.Level) bool {
	logger.SetLevel(level)
	return logger.GetLevel() == level
}

// GetDefaultLevel returns the defaultLevel
func GetDefaultLevel() logrus.Level {
	return defaultLevel
}

// GetLogger returns the pre-configured loggers
func GetLogger(level logrus.Level, logFile string) *logrus.Logger {
	once.Do(func() {
		logger, _ = getInstance(level, logFile)
	})
	if level != logger.GetLevel() {
		logger.SetLevel(level)
	}
	return logger
}

func getInstance(level logrus.Level, logFile string) (*logrus.Logger, error) {
	logrus.SetLevel(level)

	userLogger := logrus.New()
	//userLogger.SetReportCaller(true)
	userLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors:          true,
		DisableLevelTruncation: true,
		FullTimestamp:          true,
		/*CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return f.Func.Name(), "test"
		},*/
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
