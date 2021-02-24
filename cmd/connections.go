package cmd

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	vaultApi "github.com/hashicorp/vault/api"

	"github.com/cezmunsta/ssh_ms/log"
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
)

// listConnections from Vault
func listConnections(vc *vaultApi.Client) bool {
	connections, err := getConnections(vc)

	if err != nil {
		log.Panic("Unable to list connections:", err)
	}

	if len(connections) == 0 {
		fmt.Println("no available connections")
		return true
	}
	for i, s := range connections {
		m := " "
		if math.Mod(float64(i), 3) == 0 && i > 0 {
			m += "\n"
		}
		fmt.Print(s, m)
	}
	fmt.Println("")
	return true
}

// getRawConnection retrieves the secret from Vault
func getRawConnection(vc *vaultApi.Client, key string) (*vaultApi.Secret, error) {
	secret, err := vaultHelper.ReadSecret(vc, fmt.Sprintf("%s/%s", SecretPath, key))

	if err != nil || secret == nil {
		log.Warning("Unable to find connection for:", key)
		return nil, errors.New("No match found")
	}
	return secret, nil
}

// getConnections from Vault
func getConnections(vc *vaultApi.Client) ([]string, error) {
	var connections []string
	secrets, err := vaultHelper.ListSecrets(vc, SecretPath)

	if err != nil {
		log.Panic("Unable to get connections:", err)
	} else if secrets == nil || secrets.Data["keys"] == nil {
		return nil, errors.New("No data returned")
	}

	switch reflect.TypeOf(secrets.Data["keys"]).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(secrets.Data["keys"])
		for i := 0; i < s.Len(); i++ {
			connections = append(connections, fmt.Sprintf("%s", s.Index(i)))
		}
	}
	return connections, nil
}
