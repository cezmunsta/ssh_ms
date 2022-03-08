package ssh

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/cezmunsta/ssh_ms/config"
)

var (
	conn      Connection
	dummyArgs map[string]interface{}
	names     = []string{"first", "firstname.lastname", "firstnamelastname"}
)

func init() {
	for marker := range Placeholders {
		names = append(names, marker)
	}

	conn = Connection{
		HostName: "localhost",
		User:     "dummy",
		Port:     uint16(29022),
	}
	dummyArgs = map[string]interface{}{
		"HostName": conn.HostName,
		"User":     conn.User,
		"Port":     fmt.Sprintf("%d", conn.Port),
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

func TestControlPath(t *testing.T) {
	cfg := config.GetConfig()

	// Existing socket for ControlPath
	expected := fmt.Sprintf("cp_%s_%s_%d", conn.User, conn.HostName, conn.Port)
	rawCp := fmt.Sprintf("%s/%s", cfg.StoragePath, expected)

	if err := ioutil.WriteFile(rawCp, []byte("dummy"), 0o640); err != nil {
		t.Fatalf("expected: a dummy file to be created, got: %v", err)
	}

	conn.BuildConnection(dummyArgs, "dummy", conn.User)
	if !strings.HasSuffix(conn.ControlPath, expected) {
		t.Fatalf("expected: %v, got: %v", expected, conn.ControlPath)
	}
	os.Remove(rawCp)

	// New socket for ControlPath
	conn.BuildConnection(dummyArgs, "dummy", conn.User)
	expected = fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("cp_%s_%s_%d", conn.User, conn.HostName, conn.Port))))
	if !strings.HasSuffix(conn.ControlPath, expected) {
		t.Fatalf("expected: %v, got: %v", expected, conn.ControlPath)
	}
}

func TestConnection(t *testing.T) {
	cfg := config.GetConfig()

	// Test default connections
	conn.BuildConnection(dummyArgs, "dummy", conn.User)
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
	cfg.CustomLocalForward = "9998,9999"
	conn = Connection{
		HostName: "localhost",
		User:     "dummy",
		Port:     uint16(29022),
	}
	totalCustom := len(strings.Split(cfg.CustomLocalForward, ","))
	conn.BuildConnection(dummyArgs, "dummy", conn.User)
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
	if err := ioutil.WriteFile(conn.ControlPath, []byte(""), 0o600); err != nil {
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

	// Test SendEnv
	if conn.SendEnv != "" {
		t.Fatal("SendEnv is not an empty string")
	}
	if strings.Contains(conn.Cache.Config, "SendEnv") {
		t.Fatalf("expected: SendEnv to be absent, got: %v", conn.Cache.Config)
	}
	conn = Connection{
		HostName: "localhost",
		User:     "dummy",
		Port:     uint16(29022),
	}
	dummyArgs["SendEnv"] = "USER"
	conn.BuildConnection(dummyArgs, "dummy", conn.User)
	if conn.SendEnv != "USER" {
		t.Fatalf("expected: USER for SendEnv, got: %v", conn.SendEnv)
	}
	if !strings.Contains(conn.Cache.Config, "SendEnv USER") {
		t.Fatalf("expected: SendEnv USER to be present, got: %v", conn.Cache.Config)
	}
}
