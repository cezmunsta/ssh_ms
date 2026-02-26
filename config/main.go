package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	vaultApi "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

// Settings contains the configuration details
type Settings struct {
	LogLevel                                                                         logrus.Level
	Debug, RenewWarningOptOut, Simulate, StoredToken, Verbose, Version, VersionCheck bool
	ConfigComment, ConfigMotd, EnvSSHDefaultUsername, EnvSSHIdentityFile,
	CustomLocalForward, EnvSSHUsername, EnvVaultAddr, NameSpace, SecretPath, Show, StoragePath, User, VaultAddr, VaultToken, VaultAPIVersion, VaultSDKVersion string
	ServiceMap          map[string]string
	UndesiredInterfaces []string
}

var (
	once     sync.Once
	settings Settings

	// EnvBasePath is the parent location used to prefix storage paths,
	// default value is filepath.Join(os.Getenv("HOME"), ".ssh", "cache")
	EnvBasePath string

	// EnvRenewWarningOptOut will disable warnings normally sent when a token is due to expire
	// default value is os.Getenv("SSH_MS_RENEW_WARNING_OPTOUT")
	EnvRenewWarningOptOut string

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

	portServiceMappings string
	undesiredInterfaces string

	serviceMap              = make(map[string]string)
	undesiredInterfaceNames = []string{}
)

func init() {
	if v := os.Getenv("SSH_MS_SERVICE_MAP"); v != "" {
		portServiceMappings = v
	}

	if v := os.Getenv("SSH_MS_SERVICE_MAP_DISABLED"); v == "1" {
		portServiceMappings = ""
	}

	if len(portServiceMappings) > 0 {
		for _, m := range strings.Split(portServiceMappings, ";") {
			p := strings.Split(m, ":")

			if len(p) != 2 {
				panic(fmt.Sprintf("Expected 2 items, got %d: %v", len(p), p))
			}

			serviceMap[p[0]] = p[1]
		}
	}

	if len(undesiredInterfaces) > 0 {
		undesiredInterfaceNames = strings.Split(undesiredInterfaces, ",")
	}
}

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
		renewWarningOptOut := false

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

		EnvRenewWarningOptOut = os.Getenv("SSH_MS_RENEW_WARNING_OPTOUT")
		if EnvRenewWarningOptOut == "1" || EnvRenewWarningOptOut == "yes" || EnvRenewWarningOptOut == "true" {
			renewWarningOptOut = true
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
			RenewWarningOptOut:    renewWarningOptOut,
			SecretPath:            SecretPath,
			ServiceMap:            serviceMap,
			Simulate:              false,
			StoragePath:           EnvBasePath,
			StoredToken:           false,
			UndesiredInterfaces:   undesiredInterfaceNames,
			VaultAPIVersion:       vaultAPIVersion,
			VaultSDKVersion:       vaultSDKVersion,
		}
	})
	return &settings
}
