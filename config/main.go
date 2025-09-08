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
	ServiceMap map[string]string
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
	serviceMap          = make(map[string]string)
)

func init() {
	if v := os.Getenv("SSH_MS_SERVICE_MAP"); v != "" {
		// e.g. PMM:8443;SEP:8444
		for _, m := range strings.Split(v, ";") {
			p := strings.Split(m, ":")

			if len(p) != 2 {
				panic(fmt.Sprintf("Expected 2 items, got %d: %v", len(p), p))
			}

			/*if port, err := strconv.Atoi(p[1]); err == nil {
			      serviceMap[port] = p[0]
			  } else {
			      panic(fmt.Sprintf("Expected 2 items, got %d: %v", len(p), p))
			  }*/
			serviceMap[p[0]] = p[1]
		}
	} else if v := os.Getenv("SSH_MS_SERVICE_MAP_DISABLED"); v != "1" {
		serviceMap["NGINX"] = "443"
		serviceMap["PMM"] = "8443"
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
			VaultAPIVersion:       vaultAPIVersion,
			VaultSDKVersion:       vaultSDKVersion,
		}
	})
	return &settings
}
