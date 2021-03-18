package config

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Settings contains the configuration details
type Settings struct {
	LogLevel                                       logrus.Level
	Debug, Simulate, StoredToken, Verbose, Version bool
	Show, User, VaultAddr, VaultToken              string
}

var (
	once     sync.Once
	settings Settings
)

// GetConfig returns an instance of Settings
// ensuring that only one instance is ever returned
func GetConfig() *Settings {
	once.Do(func() {
		settings = Settings{
			LogLevel:    logrus.WarnLevel,
			Simulate:    false,
			StoredToken: false,
		}
	})
	return &settings
}
