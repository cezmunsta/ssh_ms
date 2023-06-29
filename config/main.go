package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	vaultApi "github.com/hashicorp/vault/api"
	vaultVersion "github.com/hashicorp/vault/version"
	"github.com/sirupsen/logrus"
)

// Settings contains the configuration details
type Settings struct {
	LogLevel                                                     logrus.Level
	Debug, Simulate, StoredToken, Verbose, Version, VersionCheck bool
	ConfigComment, ConfigMotd, EnvSSHDefaultUsername, EnvSSHIdentityFile,
	CustomLocalForward, EnvSSHUsername, EnvVaultAddr, NameSpace, SecretPath, Show, StoragePath, User, VaultAddr, VaultToken, VaultVersion string
}

var (
	once     sync.Once
	settings Settings

	/*
		The following support overrides during builds, which can be done
		by setting ldflags, e.g.

		`-ldflags "-X github.com/cezmunsta/ssh_ms/config.EnvSSHUserName=xxx"`

	*/

	// EnvBasePath is the parent location used to prefix storage paths,
	// default value is filepath.Join(os.Getenv("HOME"), ".ssh", "cache")
	EnvBasePath string

	// EnvSSHDefaultUsername sets the default used in connections,
	// default value is os.Getenv("USER")
	EnvSSHDefaultUsername string

	// EnvSSHUsername is used to authenticate with SSH
	EnvSSHUsername = "SSH_MS_USERNAME"

	// EnvSSHIdentityFile is used for SSH authentication,
	// default value is filepath.Join("~", ".ssh", "id_ed25519")
	EnvSSHIdentityFile string

	// EnvVaultAddr is the default location for Vault,
	// default value is os.Getenv(vaultApi.EnvVaultAddress)
	EnvVaultAddr string

	// SecretPath is the location used for connection manangement
	SecretPath = "secret/ssh_ms"
)

// ToJSON returns the config in JSON format
func (s *Settings) ToJSON() string {
	data, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(data)
}

// GetConfig returns an instance of Settings
// ensuring that only one instance is ever returned
func GetConfig() *Settings {
	once.Do(func() {
		if EnvBasePath == "" {
			EnvBasePath = filepath.Join(os.Getenv("HOME"), ".ssh", "cache")
		}
		EnvBasePath = NormalizePath(EnvBasePath)
		ensureDirExists(EnvBasePath)

		if EnvSSHDefaultUsername == "" {
			EnvSSHDefaultUsername = os.Getenv("USER")
		}
		if EnvSSHIdentityFile == "" {
			EnvSSHIdentityFile = filepath.Join("~", ".ssh", "id_ed25519")
		}
		if EnvVaultAddr == "" {
			EnvVaultAddr = os.Getenv(vaultApi.EnvVaultAddress)
		}
		settings = Settings{
			ConfigComment:         "",
			ConfigMotd:            "",
			CustomLocalForward:    "",
			EnvSSHDefaultUsername: EnvSSHDefaultUsername,
			EnvSSHIdentityFile:    EnvSSHIdentityFile,
			EnvSSHUsername:        EnvSSHUsername,
			EnvVaultAddr:          EnvVaultAddr,
			LogLevel:              logrus.WarnLevel,
			NameSpace:             "",
			SecretPath:            SecretPath,
			Simulate:              false,
			StoragePath:           EnvBasePath,
			StoredToken:           false,
			VaultVersion:          vaultVersion.Version,
		}
	})
	return &settings
}
