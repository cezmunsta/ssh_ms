package vault

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cezmunsta/ssh_ms/config"
	vaultKv "github.com/hashicorp/vault-plugin-secrets-kv"
	vaultApi "github.com/hashicorp/vault/api"
	vaultHttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/vault"
)

var (
	cfg             = config.GetConfig()
	client          *vaultApi.Client
	cluster         *vault.TestCluster
	once            sync.Once
	vaultSecretPath = cfg.SecretPath
)

const (
	vaultKvVersion = "1"
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

		// Create KV V1 mount
		if err := cluster.Cores[0].Client.Sys().Mount("kv", &vaultApi.MountInput{
			Type: "kv",
			Options: map[string]string{
				"version": vaultKvVersion, // TODO: update to test version 2 later
			},
		}); err != nil {
			t.Fatal(err)
		}
		// Create Secret mount
		cluster.Cores[0].Client.Sys().Unmount("secret")
		if err := cluster.Cores[0].Client.Sys().Mount(vaultSecretPath, &vaultApi.MountInput{
			Type: "kv",
			Options: map[string]string{
				"version": vaultKvVersion, // TODO: update to test version 2 later
			},
		}); err != nil {
			t.Fatal(err)
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

	key := fmt.Sprintf("%s/%s", vaultSecretPath, "dummy")
	data := make(secretData)
	data["User"] = "dummy"

	if status, err := WriteSecret(client, key, data); err != nil || !status {
		t.Fatalf("writeSecret expected: %v, got: %v, %v", data, status, err)
	}
	if secret, err := ReadSecret(client, key); err != nil || secret.Data["User"] != data["User"] {
		t.Fatalf("ReadSecret expected: %v, got: %v, %v", data, secret, err)
	}
	if status, err := DeleteSecret(client, key); err != nil || !status {
		t.Fatalf("DeleteSecret expected: %v, got: %v, %v", data, status, err)
	}

	expires := time.Now().Add(24 * time.Hour)
	if !requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf(("requiresRenewal expected: true"))
	}
	expires = expires.Add(24 * 8 * time.Hour)
	if requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf(("requiresRenewal expected: false"))
	}
}
