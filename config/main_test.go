package config

import "testing"

func TestSettings(t *testing.T) {
	s := GetConfig()
	s.User = "testme"
	sn := GetConfig()
	if sn.User != "testme" {
		t.Fatalf("expected: %v got: %v", s.User, sn.User)
	}
}
