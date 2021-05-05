package config

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// Settings contains the configuration details
type Settings struct {
	LogLevel                                                                                     logrus.Level
	Debug, Simulate, StoredToken, Verbose, Version                                               bool
	EnvSSHDefaultUsername, EnvSSHIdentityFile, EnvSSHUsername, Show, User, VaultAddr, VaultToken string
}

var (
	once     sync.Once
	settings Settings

	/*
		The following support overrides during builds, which can be done
		by setting ldflags, e.g.

		`-ldflags "-X github.com/cezmunsta/ssh_ms/config.EnvSSHUserName=xxx"`

	*/
	// EnvSSHUsername is used to authenticate with SSH
	EnvSSHUsername = "SSH_MS_USERNAME"

	// EnvSSHIdentityFile is used for SSH authentication
	EnvSSHIdentityFile = "id_rsa"
)

// GetConfig returns an instance of Settings
// ensuring that only one instance is ever returned
func GetConfig() *Settings {
	once.Do(func() {
		settings = Settings{
			EnvSSHDefaultUsername: os.Getenv("USER"),
			EnvSSHIdentityFile:    EnvSSHIdentityFile,
			EnvSSHUsername:        os.Getenv(EnvSSHUsername),
			LogLevel:              logrus.WarnLevel,
			Simulate:              false,
			StoredToken:           false,
		}
	})
	return &settings
}
