package cmd

import (
	"strings"
	"testing"

	"github.com/cezmunsta/ssh_ms/ssh"
)

func TestGetConnections(t *testing.T) {
	if cl, err := getConnections(getVaultClientWithEnv(env)); err != nil {
		t.Fatalf("expected: a connection list got: '%s'", err)
	} else {
		t.Logf("connections found: '%s'", cl)
	}
}

func TestGetRawConnection(t *testing.T) {
	if cn, err := getRawConnection(getVaultClientWithEnv(env), lookupKey); err != nil || cn == nil {
		t.Fatalf("expected: connection data got: '%v', err '%s'", cn, err)
	}
}

func TestCache(t *testing.T) {

}

func TestShowConnection(t *testing.T) {
	cn, err := getRawConnection(getVaultClientWithEnv(env), lookupKey)

	if err != nil || cn == nil {
		t.Fatalf("expected: connection data got: '%v', err '%s'", cn, err)
	}

	sshClient := ssh.Connection{}
	sshClient.BuildConnection(cn.Data, lookupKey, "dummy")
	config := sshClient.Cache.Config

	if !strings.Contains(config, lookupKey) {
		t.Fatalf("expected: got: ")
	}
}
