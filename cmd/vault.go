package cmd

import (
	vaultApi "github.com/hashicorp/vault/api"

	"github.com/cezmunsta/ssh_ms/log"
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
)

// getVaultClient by authenticating using flags
func getVaultClient() *vaultApi.Client {
	env := vaultHelper.UserEnv{
		Addr:     cfg.VaultAddr,
		Token:    cfg.VaultToken,
		Simulate: cfg.Simulate,
	}
	return getVaultClientWithEnv(env)
}

// getVaultClientWithEnv by authenticating using UserEnv
func getVaultClientWithEnv(env vaultHelper.UserEnv) *vaultApi.Client {
	if cfg.Verbose {
		log.Debug("Vault Address:", env.Addr)
		log.Debug("Simulate:", cfg.Simulate)
	}
	return vaultHelper.Authenticate(env, cfg.StoredToken)
}
