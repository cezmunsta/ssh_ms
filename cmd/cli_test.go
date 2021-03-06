package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
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
			//t.Fatalf("expected: '%s' got: '%s'", string(vb), string(out))
			continue
		}
	}
}

func TestCheckVersion(t *testing.T) {
	lines := checkVersion()
	latestVersion := "You are using the latest version"

	if len(lines) != 1 {
		t.Fatalf("expected: 1 line, got: %v", lines)
	}

	if fmt.Sprintf("%s", strings.Join(lines[0], " ")) != latestVersion {
		t.Fatalf("expected: %s, got: %v", latestVersion, lines)
	}

	Version = "foo"
	lines = checkVersion()
	if len(lines) == 1 {
		t.Fatalf("expected: 2 lines, got: %v", lines)
	}
}
