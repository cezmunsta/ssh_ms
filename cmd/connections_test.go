package cmd

import (
	"testing"
)

func TestGetConnections(t *testing.T) {
	if cl, err := getConnections(getVaultClientWithEnv(env)); err != nil {
		t.Fatalf("expected: a connection list got: '%s'", err)
	} else {
		t.Logf("connections found: '%s'", cl)
	}
}

func TestGetRawConnection(t *testing.T) {
	if cn, err := getRawConnection(getVaultClientWithEnv(env), "ceri"); err != nil || cn == nil {
		t.Fatalf("expected: connection data got: '%v', err '%s'", cn, err)
	}
}
