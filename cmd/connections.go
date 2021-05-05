package cmd

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	vaultApi "github.com/hashicorp/vault/api"

	"github.com/cezmunsta/ssh_ms/log"
	"github.com/cezmunsta/ssh_ms/ssh"
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
)

type secretData map[string]interface{}

// listConnections from Vault
func listConnections(vc *vaultApi.Client) bool {
	connections, err := getConnections(vc)

	if err != nil {
		fmt.Println("no available connections")
		return false
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

// showConnection details suitable for use with ssh_config
func showConnection(vc *vaultApi.Client, key string) bool {
	log.Debugf("showConnection %v", key)
	conn, err := getRawConnection(vc, key)

	if err != nil {
		log.Debug("Unable to show connection", key)
		return false
	}

	if conn.Data["ConfigComment"] != "" {
		fmt.Println("#", conn.Data["ConfigComment"])
	}

	sshClient := ssh.Connection{}
	sshArgs := sshClient.BuildConnection(conn.Data, key, cfg.User)
	config := sshClient.Cache.Config

	log.Info("SSH cmd:", sshArgs)

	fmt.Println(config)
	return true
}

// printConnection details suitable for use on the command line
func printConnection(vc *vaultApi.Client, key string) bool {
	log.Debugf("printConnection %v", key)
	conn, err := getRawConnection(vc, key)

	if err != nil {
		log.Debug("Unable to print connection", key)
		return false
	}

	sshClient := ssh.Connection{}
	sshArgs := sshClient.BuildConnection(conn.Data, key, cfg.User)

	fmt.Printf("ssh %v\n", strings.Join(sshArgs, " "))
	return true
}

// writeConnection creates a new entry, or updates an existing one
func writeConnection(vc *vaultApi.Client, key string, args []string) bool {
	log.Debugf("writeConnection %v", key)
	conn, err := getRawConnection(vc, key)
	secret := make(secretData)

	if err != nil {
		// New connection
		for i := 0; i < len(args); i++ {
			s := strings.Split(args[i], "=")
			secret[s[0]] = s[1]
		}
	} else {
		// Existing connection
		log.Warning("Updating an existing connection is WIP")
		log.Debug("writeConnection found: ", conn)
		return false
	}

	secret["ConfigComment"] = cfg.ConfigComment

	if cfg.Simulate {
		return true
	}

	status, err := vaultHelper.WriteSecret(vc, fmt.Sprintf("%s/%s", SecretPath, key), secret)
	if err != nil {
		log.Errorf("Failed to write '%v': %v", key, err)
		return false
	}
	return status
}

// deleteConnection removes an entry from Vault
func deleteConnection(vc *vaultApi.Client, key string) bool {
	log.Debugf("deleteConnection %v", key)
	_, err := getRawConnection(vc, key)

	if err != nil {
		log.Debug("Unable to retrieve connection", key)
		return false
	}

	status, err := vaultHelper.DeleteSecret(vc, fmt.Sprintf("%s/%s", SecretPath, key))
	if err != nil {
		log.Warning("Unable to delete connection", key)
		return false
	}
	return status
}

// getRawConnection retrieves the secret from Vault
func getRawConnection(vc *vaultApi.Client, key string) (*vaultApi.Secret, error) {
	secret, err := vaultHelper.ReadSecret(vc, fmt.Sprintf("%s/%s", SecretPath, key))

	if err != nil || secret == nil {
		log.Warning("Unable to find connection for: ", key)
		return nil, errors.New("no match found")
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
		return nil, errors.New("no data returned")
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
