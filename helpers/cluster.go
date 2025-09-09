package helpers

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	vaultApi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/helper/testcluster/docker"

	"github.com/cezmunsta/ssh_ms/config"
)

const (
	DummyComment = "This is a comment"
	DummyHost    = "dummy"
	DummyMotd    = "This is the motd"
)

var (
	cfg              = config.GetConfig()
	client           *vaultApi.Client
	cluster          *docker.DockerCluster
	once             sync.Once
	vaultSecretPaths = []string{cfg.SecretPath, cfg.SecretPath + "_v2"}

	DummyUser = cfg.EnvSSHDefaultUsername
)

// GetVaultSecretPaths is a helper to return the paths used in the test cluster
func GetVaultSecretPaths() []string {
	return vaultSecretPaths
}

// GetDummyCluster is a helper for integration testing with a real cluster.
// It returns a Vault cluster and client
func GetDummyCluster(t *testing.T) (*docker.DockerCluster, *vaultApi.Client) {
	once.Do(func() {
		ctx := context.Background()
		timeout, cancel := context.WithTimeout(ctx, time.Second*60)

		defer cancel()

		clusterOpts := docker.DefaultOptions(t)
		clusterOpts.ClusterOptions.NumCores = 1
		opts := &docker.DockerClusterOptions{
			ImageRepo:      "hashicorp/vault",
			ImageTag:       "latest",
			ClusterOptions: clusterOpts.ClusterOptions,
			DisableMlock:   true,
		}
		cluster = docker.NewTestDockerCluster(t, opts)

		client = cluster.Nodes()[0].APIClient()
		_, err := client.Logical().Read("sys/storage/raft/configuration")
		if err != nil {
			t.Fatal(err)
		}

		client.Sys().Unmount("secret")
		dummyData := map[string]interface{}{
			"HostName":      DummyHost,
			"User":          DummyUser,
			"ConfigComment": DummyComment,
			"ConfigMotd":    DummyMotd,
		}
		for _, secretPath := range vaultSecretPaths {
			version := "1"
			if strings.HasSuffix(secretPath, "_v2") {
				version = "2"
			}

			if err := client.Sys().Mount(secretPath, &vaultApi.MountInput{
				Type: "kv",
				Options: map[string]string{
					"version": version,
				},
			}); err != nil {
				t.Fatal(err)
			}

			if version == "1" {
				if err := client.KVv1(secretPath).Put(timeout, DummyHost, dummyData); err != nil {
					t.Fatalf("failed to create dummy entry for v1: %v", err)
				}
			} else {
				if _, err := client.KVv2(secretPath).Put(timeout, DummyHost, dummyData); err != nil {
					t.Fatalf("failed to create dummy entry for v2: %v", err)
				}
			}
		}

	})
	return cluster, client
}
