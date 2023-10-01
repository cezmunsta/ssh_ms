package cmd

import (
	"testing"

	"github.com/cezmunsta/ssh_ms/helpers"
	vaultApi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/helper/testcluster/docker"
)

var (
	client    *vaultApi.Client
	cluster   *docker.DockerCluster
	lookupKey = helpers.DummyHost
)

const (
	dummyComment = helpers.DummyComment
	dummyMotd    = helpers.DummyMotd
)

func getDummyCluster(t *testing.T) (*docker.DockerCluster, *vaultApi.Client) {
	cluster, client = helpers.GetDummyCluster(t)
	return cluster, client
}

/*
func generateDummyData(t *testing.T, frag string) {
	for _, sp := range strings.Split(vaultSecretPath, ",") {
		key := fmt.Sprintf("%s/%s", sp, frag)
		data := make(secretData)

		data["User"] = frag
		data["ConfigComment"] = dummyComment
		data["ConfigMotd"] = dummyMotd

		if status, err := vaultHelper.WriteSecret(client, key, data); err != nil || !status {
			t.Fatalf("writeSecret expected: %v, got: %v, %v", data, status, err)
		}
	}
}
*/
