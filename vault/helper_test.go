package vault

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	vaultKv "github.com/hashicorp/vault-plugin-secrets-kv"
	vaultApi "github.com/hashicorp/vault/api"
	vaultHttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/vault"
)

var (
	client           *vaultApi.Client
	cluster          *vault.TestCluster
	once             sync.Once
	vaultSecretPaths = []string{cfg.SecretPath, cfg.SecretPath + "_v2"}
)

const (
	vaultTestToken = "iamadummytoken"
)

func GetDummyCluster(t *testing.T) (*vault.TestCluster, *vaultApi.Client) {
	once.Do(func() {
		cluster = vault.NewTestCluster(t, &vault.CoreConfig{
			DevToken: vaultTestToken,
			LogicalBackends: map[string]logical.Factory{
				"kv": vaultKv.Factory,
			},
		}, &vault.TestClusterOptions{
			HandlerFunc: vaultHttp.Handler,
		})
		cluster.Start()

		// Create Secret mounts
		cluster.Cores[0].Client.Sys().Unmount("secret")
		for _, secretPath := range vaultSecretPaths {
			version := "1"
			if strings.HasSuffix(secretPath, "_v2") {
				version = "2"
			}

			if err := cluster.Cores[0].Client.Sys().Mount(secretPath, &vaultApi.MountInput{
				Type: "kv",
				Options: map[string]string{
					"version": version,
				},
			}); err != nil {
				t.Fatal(err)
			}
		}

		core := cluster.Cores[0].Core
		vault.TestWaitActive(t, core)
		client = cluster.Cores[0].Client
	})
	return cluster, client
}

func TestHelpers(t *testing.T) {
	cluster, client := GetDummyCluster(t)
	defer cluster.Cleanup()

	for _, secretPath := range vaultSecretPaths {
		key := fmt.Sprintf("%s/%s", secretPath, "dummy")
		data := make(secretData)
		data["User"] = "dummy"

		if status, err := WriteSecret(client, key, data); err != nil || !status {
			t.Fatalf("WriteSecret expected: %v, got: %v, %v", data, status, err)
		}

		if secret, err := ReadSecret(client, key); err != nil || secret["User"] != data["User"] {
			t.Fatalf("ReadSecret expected: %v, got: %v, %v", data, secret, err)
		}

		if status, err := DeleteSecret(client, key); err != nil || !status {
			t.Fatalf("DeleteSecret expected: %v, got: %v, %v", data, status, err)
		}
	}

	for _, secretPath := range vaultSecretPaths {
		if secrets, err := ListSecrets(client, secretPath); err != nil || len(secrets) > 0 {
			t.Fatalf("ListSecrets expected no errors nor secrets, got: %v, %v", secrets, err)
		}
	}

	expires := time.Now().Add(24 * time.Hour)
	if !requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf("requiresRenewal expected: true")
	}

	expires = expires.Add(24 * 8 * time.Hour)
	if requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf("requiresRenewal expected: false")
	}

	expires = time.Now().Add(-time.Hour)
	cfg.RenewWarningOptOut = true
	if requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf("requiresRenewal expected: false")
	}
}
