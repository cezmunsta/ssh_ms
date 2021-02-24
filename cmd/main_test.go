package cmd

import (
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
	vaultApi "github.com/hashicorp/vault/api"
)

var (
	vc  *vaultApi.Client
	env = vaultHelper.UserEnv{Addr: "http://127.0.0.1:8200", Token: "myroot", Simulate: false}
)
