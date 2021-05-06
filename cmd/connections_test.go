package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/cezmunsta/ssh_ms/config"
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
	cfg := config.GetConfig()
	key := "dummy"

	if cp := getCachePath(key); !strings.HasSuffix(cp, key+".json") {
		t.Fatalf("expected: path ending in dummy.json, got: %v", cp)
	}

	cfg.StoragePath = "/tmp/ssh_ms_cache"
	defer os.RemoveAll(cfg.StoragePath)

	if cp := getCachePath("dummy"); !strings.HasPrefix(cp, cfg.StoragePath) {
		t.Fatalf("expected: path starting with %v, got: %v", cfg.StoragePath, cp)
	}

	data := map[string]interface{}{
		"dummy": true,
	}
	if _, err := saveCache(key, data); err != nil {
		t.Fatalf("expected: no error, got: %v", err)
	}

	if cd, err := getCache(key); err != nil || len(cd) != len(data) {
		t.Fatalf("expected: %v, got: %v", data, cd)
	}

	if status, err := expireCache(key); err != nil || status == true {
		t.Fatalf("expected: false, nil, got: %v, %v", status, err)
	}
	if status, err := removeCache(key); err != nil {
		t.Fatalf("expected: true, nil, got: %v, %v", status, err)
	}
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
