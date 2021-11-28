package vault

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/cezmunsta/ssh_ms/log"
)

type secretData map[string]interface{}

// UserEnv contains settings from the ENV
type UserEnv struct {
	Addr, Token string
	Simulate    bool
}

var RenewThreshold = "168h"

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
		storedToken := ""
		if read, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".vault-token")); err != nil {
			log.Fatalf("Unable to find existing session, please login using vault")
		} else {
			storedToken = string(read)
		}
		client.SetToken(storedToken)
		storedToken = ""
	}

	if err != nil {
		log.Debug(api.EnvVaultAddress, e.Addr)
		log.Debug("Client address", client.Address())
		log.Fatal(err)
	}

	client.Auth()

	if lookupSelf, err := client.Auth().Token().LookupSelf(); err == nil {
		if requiresRenewal(lookupSelf.Data) {
			log.Warningf("Token will expire at: %v", lookupSelf.Data["expire_time"])
		}
	}
	return client
}

func requiresRenewal(d map[string]interface{}) bool {
	log.Debugf("Checking data: %v", d)
	if val, ok := d["renewable"]; ok && !val.(bool) {
		return false
	}
	if val, ok := d["expire_time"]; ok && val != nil {
		t, _ := time.Parse(time.RFC3339, val.(string))
		th, _ := time.ParseDuration(RenewThreshold)
		if time.Now().Add(th).Before(t) {
			return false
		}
	}
	return true
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
	sanitisedData := make(secretData)
	keyLookup := map[string]interface{}{
		"hostname":            "HostName",
		"port":                "Port",
		"user":                "User",
		"localforward":        "LocalForward",
		"identityfile":        "IdentityFile",
		"identitiesonly":      "IdentitiesOnly",
		"proxyjump":           "ProxyJump",
		"sendenv":             "SendEnv",
		"serveraliveinterval": "ServerAliveInterval",
		"serveralivecountmax": "ServerAliveCountMax",
		"cache":               "Cache",
		"configcomment":       "ConfigComment",
		"configmotd":          "ConfigMotd",
		"expires":             "Expires",
		"forwardagent":        "ForwardAgent",
	}

	for k, v := range data {
		opt := ""
		lk := strings.ToLower(k)
		if val, ok := keyLookup[lk]; ok {
			opt = val.(string)
		} else {
			log.Warning("Unknown option received: ", k)
			continue
		}
		sanitisedData[opt] = v
	}
	if _, err := c.Logical().Write(key, sanitisedData); err != nil {
		return false, err
	}
	return true, nil
}
