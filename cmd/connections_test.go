package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cezmunsta/ssh_ms/config"
	"github.com/cezmunsta/ssh_ms/ssh"
)

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.StoragePath = "/tmp/ssh_ms_cache"
	code := m.Run()
	defer cluster.Cleanup()
	os.RemoveAll(cfg.StoragePath)
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

	if _, err := getRawConnection(client, getLockName(lookupKey)); err == nil || fmt.Sprintf("%s", err) != "no lock found" {
		t.Fatalf("expected: no lock found got: '%v'", err)
	}

	for _, cmd := range []string{"connect", "write", "update", "delete", "list", "search", "show", "print", "purge"} {
		currentCommand = cmd
		switch cmd {
		case "write":
			if _, err := getRawConnection(client, "thisisnotavaliditem"); err == nil || fmt.Sprintf("%s", err) != "no lock found" {
				t.Fatalf("expected: 'no lock found', got: %v", err)
			}
		default:
			if _, err := getRawConnection(client, "thisisnotavaliditem"); err == nil || fmt.Sprintf("%s", err) != "no match found" {
				t.Fatalf("expected: 'no match found' present, got: %v", err)
			}
		}
	}
}

func TestCache(t *testing.T) {
	cfg := config.GetConfig()
	key := lookupKey
	data := map[string]interface{}{
		key: true,
	}

	// Test core functionality
	if cp := getCachePath(key); !strings.HasSuffix(cp, key+".json") {
		t.Fatalf("expected: path ending in dummy.json, got: %v", cp)
	}

	if cp := getCachePath(key); !strings.HasPrefix(cp, cfg.StoragePath) {
		t.Fatalf("expected: path starting with %v, got: %v", cfg.StoragePath, cp)
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

	// Test purging
	purgeForce = true
	purgeCache()
	if dirInfo, err := os.Stat(cfg.StoragePath); err == nil {
		t.Fatalf("expected: %v to be absent, got: %v", cfg.StoragePath, dirInfo)
	}

	// Test population
	_, client := getDummyCluster(t)
	populateCache(client)
	if _, err := getCache(key); err != nil {
		t.Fatalf("expected: cache file to be present, got: %v", err)
	}
}

func TestShowConnection(t *testing.T) {
	_, client := getDummyCluster(t)
	cn, err := getRawConnection(client, lookupKey)

	if err != nil || cn == nil {
		t.Fatalf("expected: connection data got: '%v', err '%s'", cn, err)
	}

	sshClient := ssh.Connection{}
	sshClient.BuildConnection(cn, lookupKey, "dummy")
	config := sshClient.Cache.Config

	if !strings.Contains(config, lookupKey) {
		t.Fatalf("expected: got: ")
	}
}

func TestPrepareConnection(t *testing.T) {
	_, client := getDummyCluster(t)
	cn, err := getRawConnection(client, lookupKey)

	if err != nil || cn == nil {
		t.Fatalf("expected: connection data got: '%v', err '%s'", cn, err)
	}

	removeCache(lookupKey)
	_, sshClient, configComment, configMotd := prepareConnection(client, []string{lookupKey})

	if sshClient.User != lookupKey {
		t.Fatalf("expected user '%v', got '%v'", lookupKey, sshClient.User)
	}

	if configComment != dummyComment {
		t.Fatalf("expected comment '%v', got '%v'", dummyComment, configComment)
	}

	if !strings.Contains(configMotd, dummyMotd) {
		t.Fatalf("expected motd to contain '%v', got '%v'", dummyMotd, configMotd)
	}
}
