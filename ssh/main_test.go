package ssh

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/cezmunsta/ssh_ms/config"
)

var (
	names = []string{"first", "firstname.lastname", "firstnamelastname"}
)

func init() {
	for marker := range Placeholders {
		names = append(names, marker)
	}
}

func TestGenerateUserName(t *testing.T) {
	var userNames []userName

	for _, n := range names {
		user := userName{}
		if _, err := user.generateUserName(n); err != nil {
			t.Fatalf("expected: userName{} got: %v", err)
		}
		userNames = append(userNames, user)
	}

	if len(userNames) > 0 {
		user := userNames[0]
		if _, err := user.generateUserName(names[0]); err == nil {
			t.Fatalf("expected: an error got: %v", user)
		}
	}
}

func TestMarshal(t *testing.T) {
	for _, n := range names {
		user := userName{FullName: n}
		jsonUser, err := user.doMarshal()
		if err != nil {
			t.Fatalf("expected: %v as JSON got: %v with error %v", user, jsonUser, err)
		}

		newUser, err := user.doUnmarshal(jsonUser)
		if err != nil {
			t.Fatalf("expected: %v as userName got: %v with error %v", jsonUser, newUser, err)

		}

		if user.FullName != newUser.FullName {
			t.Fatalf("expected: %v got: %v", user.FullName, newUser.FullName)
		}

		if keyedUser, err := user.doUnmarshalToKeys(jsonUser); err != nil || keyedUser["FullName"] != user.FullName {
			t.Fatalf("expected: %v got: %v", user.FullName, keyedUser)
		}
	}
}

func TestRewriteUsername(t *testing.T) {
	for _, n := range names {
		user := userName{}
		if _, err := user.generateUserName(n); err != nil {
			t.Fatalf("expected: userName{} got: %v", err)
		}
		user.rewriteUsername("ceri.williams")
		if strings.Contains(user.FullName, "@") && !strings.Contains(n, "@") {
			t.Fatalf("expected: templates to be parsed got: %v", user.FullName)
		}
	}
}

func TestConnection(t *testing.T) {
	conn := Connection{
		HostName: "localhost",
		User:     "dummy",
		Port:     uint16(29022),
	}
	// Test default connections
	conn.BuildConnection(map[string]interface{}{}, "dummy", conn.User)
	expected := fmt.Sprintf("cp_%s_%s_%d", conn.User, conn.HostName, conn.Port)
	if !strings.HasSuffix(conn.ControlPath, expected) {
		t.Fatalf("expected: %v, got: %v", expected, conn.ControlPath)
	}
	if len(conn.LocalForward) != 2 {
		t.Fatalf("expected: 2 LocalFoward rules, got: %v", conn.LocalForward)
	}
	for _, lf := range conn.LocalForward {
		switch lf.RemotePort {
		case 443, 8443:
			continue
		default:
			t.Fatalf("unexpected remote port: %v", lf.RemotePort)
		}
	}

	// Test default ForwardAgent is set to "no"
	if conn.ForwardAgent != "no" {
		t.Fatalf("ForwardAgent is not set to 'no'")
	}

	// Test custom LocalForward rules
	cfg := config.GetConfig()
	cfg.CustomLocalForward = "9998,9999"
	conn = Connection{
		HostName: "localhost",
		User:     "dummy",
		Port:     uint16(29022),
	}
	totalCustom := len(strings.Split(cfg.CustomLocalForward, ","))
	conn.BuildConnection(map[string]interface{}{}, "dummy", conn.User)
	if len(conn.LocalForward) != totalCustom {
		t.Fatalf("expected: %v LocalFoward rules, got: %v", totalCustom, conn.LocalForward)
	}
	for _, lf := range conn.LocalForward {
		switch lf.RemotePort {
		case 9998, 9999:
			continue
		default:
			t.Fatalf("unexpected remote port: %v", lf.RemotePort)
		}
	}

	// Test loading custom LocalForward ports from cache
	if err := ioutil.WriteFile(conn.ControlPath, []byte(""), 0600); err != nil {
		t.Fatal("failed to write dummy ControlPath:", conn.ControlPath, err)
	}
	setPortForwarding(&conn)
	for _, lf := range conn.LocalForward {
		switch lf.RemotePort {
		case 9998, 9999:
			continue
		default:
			t.Fatalf("unexpected remote port: %v", lf.RemotePort)
		}
	}
}
