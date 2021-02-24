package cmd

import (
	vaultApi "github.com/hashicorp/vault/api"

	"github.com/cezmunsta/ssh_ms/log"
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
)

// getVaultClient by authenticating using flags
func getVaultClient() *vaultApi.Client {
	env := vaultHelper.UserEnv{Addr: flags.Addr, Token: flags.Token, Simulate: flags.Simulate}
	return getVaultClientWithEnv(env)
}

// getVaultClientWithEnv by authenticating using UserEnv
func getVaultClientWithEnv(env vaultHelper.UserEnv) *vaultApi.Client {
	if flags.Verbose {
		log.Debug("Vault Address:", env.Addr)
		log.Debug("Simulate:", flags.Simulate)
	}
	return vaultHelper.Authenticate(env, flags.StoredToken)
}
