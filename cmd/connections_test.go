package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/cezmunsta/ssh_ms/ssh"
)

func TestMain(m *testing.M) {
	code := m.Run()
	defer cluster.Cleanup()
	os.Exit(code)
}

func TestGetVaultClient(t *testing.T) {
	_, client = getDummyCluster(t)

	if len(client.Token()) == 0 {
		t.Fatalf("expected: a non-zero length token got: a zero length token")
	}
}

func TestGetConnections(t *testing.T) {
	_, client := getDummyCluster(t)

	if cl, err := getConnections(client); err != nil {
		t.Fatalf("expected: a connection list got: '%s'", err)
	} else {
		t.Logf("connections found: '%s'", cl)
	}
}

func TestGetRawConnection(t *testing.T) {
	_, client := getDummyCluster(t)

	if cn, err := getRawConnection(client, lookupKey); err != nil || cn == nil {
		t.Fatalf("expected: connection data got: '%v', err '%s'", cn, err)
	}
}

/*func TestCache(t *testing.T) {
	cfg := config.GetConfig()
	key := lookupKey

	if cp := getCachePath(key); !strings.HasSuffix(cp, key+".json") {
		t.Fatalf("expected: path ending in dummy.json, got: %v", cp)
	}

	cfg.StoragePath = "/tmp/ssh_ms_cache"
	defer os.RemoveAll(cfg.StoragePath)

	if cp := getCachePath(key); !strings.HasPrefix(cp, cfg.StoragePath) {
		t.Fatalf("expected: path starting with %v, got: %v", cfg.StoragePath, cp)
	}

	data := map[string]interface{}{
		key: true,
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
}*/

func TestShowConnection(t *testing.T) {
	_, client := getDummyCluster(t)
	cn, err := getRawConnection(client, lookupKey)

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
