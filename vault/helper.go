package vault

import (
	"context"
	"fmt"
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

// RenewThreshold is used to compare against the token expiration time
var RenewThreshold = "168h"

const (
	apiTimeout           = time.Second * 60
	errHasMetadataSuffix = "metadata is a reserved word"
	errNoMatchFound      = "no match found"
)

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
		if read, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".vault-token")); err != nil {
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
	ctx := context.Background()
	mountPath, secretName := getSplitPath(key)
	timeout, cancel := context.WithTimeout(ctx, apiTimeout)

	defer cancel()

	if ver, err := getKvVersion(c, mountPath); err == nil {
		switch ver {
		case "kv2":
			// TODO: this should be changed when adding support for revisions
			if err := c.KVv2(mountPath).DeleteMetadata(timeout, secretName); err != nil {
				return false, err
			}
		case "kv1":
			if err := c.KVv1(mountPath).Delete(timeout, secretName); err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// ListSecrets reads the list of secrets/data under a path in Vault
// c : Vault client
// path : path to secret/data in Vault
func ListSecrets(c *api.Client, path string) ([]*api.Secret, []error) {
	errors := []error{}
	paths := strings.Split(path, ",")
	secrets := []*api.Secret{}

	for _, p := range paths {
		var err error
		var secret *api.Secret
		var ver string

		ver, err = getKvVersion(c, p)
		switch ver {
		case "kv2":
			secret, err = c.Logical().List(p + "/metadata")
		case "kv1":
			secret, err = c.Logical().List(p)
		default:
			errors = append(errors, fmt.Errorf("unable to match KV version: %v", ver))
		}

		if err != nil {
			errors = append(errors, err)
		} else if secret != nil {
			secrets = append(secrets, secret)
		}
	}

	if len(errors) > 0 {
		return secrets, errors
	}

	return secrets, nil
}

// ReadSecret requests the secret/data from Vault
// c : Vault client
// key : the key for the desired secret/data
func ReadSecret(c *api.Client, key string) (map[string]interface{}, error) {
	ctx := context.Background()
	mountPath, secretName := getSplitPath(key)
	timeout, cancel := context.WithTimeout(ctx, apiTimeout)

	defer cancel()

	if ver, err := getKvVersion(c, mountPath); err == nil {
		switch ver {
		case "kv2":
			if secret, _ := c.KVv2(mountPath).Get(timeout, secretName); secret != nil {
				return secret.Data, nil
			}
		case "kv1":
			if secret, _ := c.KVv1(mountPath).Get(timeout, secretName); secret != nil {
				return secret.Data, nil
			}
		}
	}

	return nil, fmt.Errorf(errNoMatchFound)
}

// WriteSecret adds a secret to Vault
// c : Vault client
// key : the key for the secret
// data : config for use when writing data
func WriteSecret(c *api.Client, key string, data map[string]interface{}) (bool, error) {
	ctx := context.Background()
	mountPath, secretName := getSplitPath(key)
	timeout, cancel := context.WithTimeout(ctx, apiTimeout)

	defer cancel()

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

	if ver, err := getKvVersion(c, mountPath); err == nil {
		switch ver {
		case "kv2":
			if _, err := c.KVv2(mountPath).Put(timeout, secretName, sanitisedData); err != nil {
				return false, err
			}
		case "kv1":
			if err := c.KVv1(mountPath).Put(timeout, secretName, sanitisedData); err != nil {
				return false, err
			}
		default:
			return false, err
		}
	}

	return true, nil
}

func getSplitPath(path string) (string, string) {
	sp := strings.Split(path, "/")

	return strings.Join(sp[0:len(sp)-1], "/"), sp[len(sp)-1]
}

func getKvVersion(c *api.Client, path string) (string, error) {
	log.Debugf("getKvVersion: %s", path)

	// Ensure that we aren't getting secret/mount/path/metadata getting
	// passed in to the check, as this is used for KV2 secrets
	if strings.HasSuffix(path, "/metadata") {
		log.Debugf("%s has metadata suffix", path)
		return "", fmt.Errorf(errHasMetadataSuffix)
	}

	secret, err := c.Logical().List(path)

	if err == nil && (secret == nil || len(secret.Warnings) == 0) {
		log.Debugf("%s is kv1", path)
		return "kv1", nil
	}

	if err == nil && secret != nil && len(secret.Warnings) > 0 {
		log.Debugf("%s is kv2", path)
		return "kv2", nil
	}

	log.Debugf("%s is unknown", path)
	return "", fmt.Errorf(errNoMatchFound)
}
