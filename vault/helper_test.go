package vault

import (
	"fmt"
	"testing"
	"time"

	"github.com/cezmunsta/ssh_ms/helpers"
)

func TestHelpers(t *testing.T) {
	cluster, client := helpers.GetDummyCluster(t)
	vaultSecretPaths := helpers.GetVaultSecretPaths()

	defer cluster.Cleanup()

	for _, secretPath := range vaultSecretPaths {
		key := fmt.Sprintf("%s/%s", secretPath, "dummy")
		data := make(secretData)
		data["User"] = "dummy"

		if status, err := WriteSecret(client, key, data); err != nil || !status {
			t.Fatalf("WriteSecret expected: %v, got: %v, %v", data, status, err)
		}

		if secret, err := ReadSecret(client, key); err != nil || secret["User"] != data["User"] {
			t.Fatalf("ReadSecret expected: %v, got: %v, %v", data, secret, err)
		}

		if status, err := DeleteSecret(client, key); err != nil || !status {
			t.Fatalf("DeleteSecret expected: %v, got: %v, %v", data, status, err)
		}
	}

	for _, secretPath := range vaultSecretPaths {
		if secrets, err := ListSecrets(client, secretPath); err != nil || len(secrets) > 0 {
			t.Fatalf("ListSecrets expected no errors nor secrets, got: %v, %v", secrets, err)
		}
	}

	expires := time.Now().Add(24 * time.Hour)
	if !requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf("requiresRenewal expected: true")
	}

	expires = expires.Add(24 * 8 * time.Hour)
	if requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf("requiresRenewal expected: false")
	}

	expires = time.Now().Add(-time.Hour)
	cfg.RenewWarningOptOut = true
	if requiresRenewal(map[string]interface{}{
		"renewable":   true,
		"expire_time": expires.Format(time.RFC3339),
	}) {
		t.Fatalf("requiresRenewal expected: false")
	}
}
