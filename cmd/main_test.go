package cmd

import (
	"os"

	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
	vaultApi "github.com/hashicorp/vault/api"
)

var (
	vc *vaultApi.Client

	env       = vaultHelper.UserEnv{Addr: os.Getenv("VAULT_ADDR"), Token: os.Getenv("VAULT_TOKEN"), Simulate: false}
	lookupKey = "test"
)
