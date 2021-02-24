package cmd

import (
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"

	vaultApi "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"

	"github.com/cezmunsta/ssh_ms/log"
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
)

// EnvBasePath is the parent location used to prefix storage paths
const EnvBasePath = "HOME"

type cmdFlags struct {
	List, Simulate, StoredToken, Verbose, Version bool
	Addr, Show, Token                             string
}

//type cmdFlags struct {
//    Expert, Purge, bool
//    Comment, StoragePath, User, Write         string
//}

var (
	rootCmd = &cobra.Command{
		Use:   "ssh_ms",
		Short: "ssh_ms connects you to your remote world",
		Long:  "ssh_ms integrates with HashiCorp Vault to store SSH configs and ease your remote life",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Name() == "help" {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if flags.Version {
				printVersion()
				os.Exit(0)
			} else {
				cmd.Usage()
				os.Exit(1)
			}
		},
	}

	listCmd = &cobra.Command{
		Use:   "list [flags]",
		Short: "List available connections",
		Long:  "Lookup available connections in Vault and list them",
		Run: func(cmd *cobra.Command, args []string) {
			listConnections(getVaultClient())
		},
	}

	flags = cmdFlags{}

	/*
		The following support overrides during builds, which can be done
		by setting ldflags, e.g.

		`-ldflags "-X github.com/cezmunsta/ssh_ms/cmd.Version=xxx"`

	*/

	// EnvSSHUsername is used to authenticate with SSH
	EnvSSHUsername = "SSH_MS_USERNAME"

	// EnvSSHIdentityFile is used for SSH authentication
	EnvSSHIdentityFile = "id_rsa"

	// EnvVaultAddr is the default location for Vault
	EnvVaultAddr = vaultApi.EnvVaultAddress

	// SecretPath is the location used for connection manangement
	SecretPath = "secret/ssh_ms"

	// Version of the code
	Version = "1.0"
)

func init() {
	rootCmd.AddCommand(
		listCmd,
	)
	rootCmd.PersistentFlags().StringVar(&flags.Addr, "vault-addr", os.Getenv(vaultApi.EnvVaultAddress), "Specify the Vault address")
	rootCmd.PersistentFlags().StringVar(&flags.Token, "vault-token", os.Getenv(vaultApi.EnvVaultToken), "Specify the Vault token")

	rootCmd.PersistentFlags().BoolVarP(&flags.StoredToken, "stored-token", "", false, "Use a stored token from 'vault login' (overrides --vault-token, auto-enabled when no token is specified)")
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Provide addition output")

	rootCmd.Flags().BoolVarP(&flags.Version, "version", "V", false, "Show the version")
}

// getVaultClient by authenticating using flags
func getVaultClient() *vaultApi.Client {
	env := vaultHelper.UserEnv{Addr: flags.Addr, Token: flags.Token, Simulate: flags.Simulate}
	return getVaultClientWithEnv(env)
}

// getVaultClientWithEnv by authenticating using UserEnv
func getVaultClientWithEnv(env vaultHelper.UserEnv) *vaultApi.Client {
	if flags.Verbose {
		log.Debug("Vault Address:", env.Addr)
		log.Debug("Simulate:", flags.Simulate)
	}
	return vaultHelper.Authenticate(env, flags.StoredToken)
}

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

// getVersion information for the application
func getVersion() [][]string {
	var lines [][]string

	if !flags.Verbose {
		lines = append(lines, []string{Version})
	} else {
		lines = append(lines, []string{"Version:", Version})
		lines = append(lines, []string{"Arch:", runtime.GOOS, runtime.GOARCH})
	}
	return lines
}

// printVersion of the application
func printVersion() {
	lines := getVersion()

	for _, line := range lines {
		fmt.Println(strings.Join(line, " "))
	}
}

// Execute the commands
func Execute() (int, error) {
	if err := rootCmd.Execute(); err != nil {
		return 1, err
	}
	return 0, nil
}
