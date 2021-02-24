package vault

import (
	"os"

	"github.com/hashicorp/vault/api"
	credToken "github.com/hashicorp/vault/builtin/credential/token"
	login "github.com/hashicorp/vault/command"

	"github.com/cezmunsta/ssh_ms/log"
)

// UserEnv contains settings from the ENV
type UserEnv struct {
	Addr, Token string
	Simulate    bool
}

// Authenticate a user with Vault
// e : user environment
// st: flag to use the stored token
func Authenticate(e UserEnv, st bool) *api.Client {
	os.Setenv(api.EnvVaultAddress, e.Addr)
	defer os.Setenv(api.EnvVaultAddress, "")

	if !st {
		os.Setenv(api.EnvVaultToken, e.Token)
	}
	defer os.Setenv(api.EnvVaultToken, "")

	os.Setenv(api.EnvVaultMaxRetries, "3")
	defer os.Setenv(api.EnvVaultMaxRetries, "")

	config := api.DefaultConfig()
	client, err := api.NewClient(config)

	if st {
		lc := &login.LoginCommand{
			BaseCommand: &login.BaseCommand{},
			Handlers: map[string]login.LoginHandler{
				"token": &credToken.CLIHandler{},
			},
		}
		th, err := lc.TokenHelper()
		if err != nil {
			log.Debug("TokenHelper:", th)
			log.Panic("Uh oh, the TokenHelper had an issue")
		}
		storedToken, err := th.Get()
		if err != nil {
			log.Debug("storedToken:", storedToken)
			log.Panic("Unable to read token from store")
		}
		client.SetToken(storedToken)
	}

	if err != nil {
		log.Debug(api.EnvVaultAddress, e.Addr)
		log.Debug("Client address", client.Address())
		log.Panic(err)
	}

	client.Auth()
	return client
}

// DeleteSecret removes a secret from Vault
func DeleteSecret(c *api.Client, key string) (bool, error) {
	if _, err := c.Logical().Delete(key); err != nil {
		return false, err
	}
	return true, nil
}

// ListSecrets reads the list of secrets/data under a path in Vault
// c : Vault client
// path : path to secret/data in Vault
func ListSecrets(c *api.Client, path string) (*api.Secret, error) {
	secret, err := c.Logical().List(path)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// ReadSecret requests the secret/data from Vault
// c : Vault client
// key : the key for the desired secret/data
func ReadSecret(c *api.Client, key string) (*api.Secret, error) {
	secret, err := c.Logical().Read(key)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// WriteSecret adds a secret to Vault
// c : Vault client
// key : the key for the secret
// data : config for use when writing data
func WriteSecret(c *api.Client, key string, data map[string]interface{}) (bool, error) {
	if _, err := c.Logical().Write(key, data); err != nil {
		return false, err
	}
	return true, nil
}
