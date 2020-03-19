package vault

import (
	"log"

	"github.com/hashicorp/vault/api"
)

// UserEnv contains settings from the ENV
type UserEnv struct {
	Addr, Token string
	Simulate    bool
}

// DeleteSecret removes a secret from Vault
func DeleteSecret(c api.Client, key string) bool {
	if _, err := c.Logical().Delete(key); err != nil {
		log.Fatal(err)
		return false
	}

	return true
}

// ListSecrets reads the list of secrets/data under a path in Vault
// c : Vault client
// path : path to secret/data in Vault
func ListSecrets(c api.Client, path string) *api.Secret {
	secret, err := c.Logical().List(path)
	if err != nil {
		log.Fatal(err)
	}
	return secret
}

// ReadSecret requests the secret/data from Vault
// c : Vault client
// key : the key for the desired secret/data
func ReadSecret(c api.Client, key string) *api.Secret {
	secret, err := c.Logical().Read(key)
	if err != nil {
		log.Fatal(err)
	}
	return secret
}

// WriteSecret adds a secret to Vault
func WriteSecret(c api.Client, key string, data map[string]interface{}) bool {
	if _, err := c.Logical().Write("secret/ssh_ms/"+key, data); err != nil {
		log.Fatal(err)
		return false
	}

	return true
}
