package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Settings contains the configuration details
type Settings struct {
	LogLevel                                       logrus.Level
	Debug, Simulate, StoredToken, Verbose, Version bool
	ConfigComment, EnvSSHDefaultUsername, EnvSSHIdentityFile,
	EnvSSHUsername, Show, StoragePath, User, VaultAddr, VaultToken string
}

var (
	once     sync.Once
	settings Settings

	/*
		The following support overrides during builds, which can be done
		by setting ldflags, e.g.

		`-ldflags "-X github.com/cezmunsta/ssh_ms/config.EnvSSHUserName=xxx"`

	*/
	// EnvBasePath is the parent location used to prefix storage paths
	EnvBasePath = filepath.Join(os.Getenv("HOME"), ".ssh", "cache")

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
			ConfigComment:         "",
			EnvSSHDefaultUsername: os.Getenv("USER"),
			EnvSSHIdentityFile:    EnvSSHIdentityFile,
			EnvSSHUsername:        os.Getenv(EnvSSHUsername),
			LogLevel:              logrus.WarnLevel,
			Simulate:              false,
			StoragePath:           EnvBasePath,
			StoredToken:           false,
		}
	})
	return &settings
}
