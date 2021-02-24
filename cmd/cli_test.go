package cmd

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
	vaultApi "github.com/hashicorp/vault/api"
)

var (
	vc  *vaultApi.Client
	env = vaultHelper.UserEnv{Addr: "http://127.0.0.1:8200", Token: "myroot", Simulate: false}
)

func TestExecute(t *testing.T) {
	var vb []byte

	cmd := rootCmd
	ver := getVersion()
	b := bytes.NewBufferString("")

	cmd.SetOut(b)
	cmd.SetArgs([]string{"--version"})
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

func TestGetVaultClient(t *testing.T) {
	vc = getVaultClientWithEnv(env)
	if len(vc.Token()) == 0 {
		t.Fatalf("expected: a non-zero length token got: a zero length token")
	}
}

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
