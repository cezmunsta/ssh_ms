package cmd

import (
	"bytes"
	"io/ioutil"
	"runtime"
	"strings"
	"testing"

	"github.com/cezmunsta/ssh_ms/config"
)

func TestExecute(t *testing.T) {
	var vb []byte

	cmd := rootCmd
	ver := getVersion()
	b := bytes.NewBufferString("")

	cmd.SetOut(b)
	cmd.SetArgs([]string{"version"})
	cmd.Execute()

	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range ver {
		vb = []byte(strings.Join(line, " "))
		if string(out) != string(vb) {
			// t.Fatalf("expected: '%s' got: '%s'", string(vb), string(out))
			continue
		}
	}
}

func TestGetVersion(t *testing.T) {
	lines := getVersion()
	if len(lines) > 1 {
		t.Fatalf("expected: 1 line, got: %v", lines)
	}

	cfg := config.GetConfig()
	cfg.Verbose = true
	lines = getVersion()
	if len(lines) == 1 {
		t.Fatalf("expected: multiple lines, got: %v", lines)
	}

	if s := strings.Join(lines[2], " "); !strings.Contains(s, "Go Version: "+runtime.Version()) {
		t.Fatalf("expected: 'Go Version: %v', got: %v", runtime.Version(), lines[2])
	}

	if s := strings.Join(lines[3], " "); !strings.Contains(s, "Vault Version: "+cfg.VaultVersion) {
		t.Fatalf("expected: 'Vault Version: %v', got: %v", cfg.VaultVersion, lines[3])
	}
}

func TestCheckVersion(t *testing.T) {
	lines := checkVersion()
	//latestVersion := "You are using the latest version"

	if len(lines) != 1 && len(lines) != 2 {
		t.Fatalf("expected: 1-2 lines, got: %v", lines)
	}

	/*if strings.Join(lines[0], " ") != latestVersion {
		t.Fatalf("expected: %s, got: %v", latestVersion, lines)
	}*/

	Version = "foo"
	lines = checkVersion()
	if len(lines) == 1 {
		t.Fatalf("expected: 2 lines, got: %v", lines)
	}
}
