package ssh

import (
	"testing"
)

var (
	names = []string{"ceri", "ceri.williams", "ceriwilliams"}
)

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
