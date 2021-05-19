package cmd

import (
	"fmt"
	"sync"
	"testing"

	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
	vaultKv "github.com/hashicorp/vault-plugin-secrets-kv"
	vaultApi "github.com/hashicorp/vault/api"
	vaultHttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/vault"
)

var (
	client          *vaultApi.Client
	cluster         *vault.TestCluster
	lookupKey       = "dummy"
	once            sync.Once
	vaultSecretPath = cfg.SecretPath
)

const (
	dummyComment = "This is a comment"
	dummyMotd    = "This is the motd"

	vaultKvVersion = "1"
	vaultTestToken = "iamadummytoken"
)

func getDummyCluster(t *testing.T) (*vault.TestCluster, *vaultApi.Client) {
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
		generateDummyData(t, lookupKey)
	})
	return cluster, client
}

func generateDummyData(t *testing.T, frag string) {
	key := fmt.Sprintf("%s/%s", vaultSecretPath, frag)
	data := make(secretData)
	data["User"] = frag
	data["ConfigComment"] = dummyComment
	data["ConfigMotd"] = dummyMotd

	if status, err := vaultHelper.WriteSecret(client, key, data); err != nil || !status {
		t.Fatalf("writeSecret expected: %v, got: %v, %v", data, status, err)
	}
}
