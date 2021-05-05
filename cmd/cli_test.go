package cmd

import (
	"bytes"
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
			t.Fatalf("expected: '%s' got: '%s'", string(vb), string(out))
		}
	}
}
