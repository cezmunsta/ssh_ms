package vault

//import (
//	"testing"
//
//	kv "github.com/hashicorp/vault-plugin-secrets-kv"
//	"github.com/hashicorp/vault/api"
//	hchttp "github.com/hashicorp/vault/http"
//	"github.com/hashicorp/vault/sdk/logical"
//	"github.com/hashicorp/vault/vault"
//)
//
//type secret struct {
//	Path           string
//	Value          string
//	ExpectedStatus bool
//}
//
//type secretData map[string]secret
//
//var client api.Client
//var dummySecret *api.Secret
//var sd secretData
//
////func createTestVault(t *testing.T) (net.Listener, *api.Client) {
////	t.Helper()
////
////	// Create an in-memory, unsealed core (the "backend", if you will).
////	core, keyShares, rootToken := vault.TestCoreUnsealed(t)
////	_ = keyShares
////
////	// Start an HTTP server for the core.
////	ln, addr := http.TestServer(t, core)
////
////	// Create a client that talks to the server, initially authenticating with
////	// the root token.
////	conf := api.DefaultConfig()
////	conf.Address = addr
////
////	client, err := api.NewClient(conf)
////	if err != nil {
////		t.Fatal(err)
////	}
////	client.SetToken(rootToken)
////
////	// Setup required secrets, policies, etc.
////	for _, v := range sd {
////		if !v.ExpectedStatus {
////			continue
////		}
////		_, err = client.Logical().Write(v.Path, map[string]interface{}{
////			"secret": v.Value,
////		})
////		if err != nil {
////			t.Fatal(err)
////		}
////	}
////
////	return ln, client
////}
//
//func createTestCluster(t *testing.T) *vault.TestCluster {
//	t.Helper()
//
//	coreConfig := &vault.CoreConfig{
//		LogicalBackends: map[string]logical.Factory{
//			"kv": kv.Factory,
//		},
//	}
//	cluster := vault.NewTestCluster(t, coreConfig, &vault.TestClusterOptions{
//		HandlerFunc: hchttp.Handler,
//	})
//	cluster.Start()
//
//	// Create KV V1 mount
//	if err := cluster.Cores[0].Client.Sys().Mount("kv", &api.MountInput{
//		Type: "kv",
//		Options: map[string]string{
//			"version": "1", // TODO: update to test version 2 later
//		},
//	}); err != nil {
//		t.Fatal(err)
//	}
//	// Create Secret mount
//	cluster.Cores[0].Client.Sys().Unmount("secret")
//	if err := cluster.Cores[0].Client.Sys().Mount("secret/ssh_ms", &api.MountInput{
//		Type: "kv",
//		Options: map[string]string{
//			"version": "1", // TODO: update to test version 2 later
//		},
//	}); err != nil {
//		t.Fatal(err)
//	}
//
//	return cluster
//}
//
////func createTestClient(t *testing.T) *api.Client {
////	cluster := createTestCluster(t)
////	defer cluster.Cleanup()
////	// Setup required secrets, policies, etc.
////	for _, v := range sd {
////		if !v.ExpectedStatus {
////			continue
////		}
////		_, err := cluster.Cores[0].Client.Logical().Write(v.Path, map[string]interface{}{
////			"secret": v.Value,
////		})
////		if err != nil {
////			t.Fatal(err)
////		}
////	}
////
////	return cluster.Cores[0].Client
////}
//
//func createTestData(c *api.Client) error {
//	// Setup required secrets, policies, etc.
//	var err error
//	sd = secretData{}
//
//	sd["present"] = secret{"secret/ssh_ms/foo", "OK", true}
//	sd["absent"] = secret{"secret/ssh_ms/bar", "NOK", false}
//
//	for _, v := range sd {
//		if !v.ExpectedStatus {
//			continue
//		}
//		_, err = c.Logical().Write(v.Path, map[string]interface{}{
//			"secret": v.Value,
//		})
//		if err != nil {
//			break
//		}
//	}
//	return err
//}
//
//func TestListSecrets(t *testing.T) {
//	//client := createTestClient(t)
//	cluster := createTestCluster(t)
//	defer cluster.Cleanup()
//	createTestData(cluster.Cores[0].Client)
//
//	for k, v := range sd {
//		dummySecret = ListSecrets(cluster.Cores[0].Client, v.Path)
//		switch v.ExpectedStatus {
//		case true:
//			if dummySecret.Data["secret"] != v.Value {
//				t.Errorf("ListSecrets %s test failed, expected: '%v', got: '%v'", k, v.Value, dummySecret.Data["secret"])
//			}
//		case false:
//			if dummySecret.Data["secret"] == v.Value {
//				t.Errorf("ListSecrets %s test failed, expected: '%v', got: '%v'", k, v.Value, dummySecret.Data["secret"])
//			}
//		}
//	}
//}
//
