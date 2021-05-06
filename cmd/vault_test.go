package cmd

import (
	"testing"
)

func TestGetVaultClient(t *testing.T) {
	vc = getVaultClientWithEnv(env)
	if len(vc.Token()) == 0 {
		t.Fatalf("expected: a non-zero length token got: a zero length token")
	}
}
