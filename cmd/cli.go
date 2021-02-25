package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	vaultApi "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
)

// EnvBasePath is the parent location used to prefix storage paths
const EnvBasePath = "HOME"

type cmdFlags struct {
	List, Simulate, StoredToken, Verbose, Version bool
	Addr, Show, Token, User                       string
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

	showCmd = &cobra.Command{
		Use:   "show CONNECTION [flags]",
		Short: "Display a connection",
		Long:  "Display the SSH config for the requested connection",
		Example: `
    ssh_ms show gateway
        `,
		Run: func(cmd *cobra.Command, args []string) {
			showConnection(getVaultClient(), args[0])
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
		showCmd,
	)
	rootCmd.PersistentFlags().StringVar(&flags.Addr, "vault-addr", os.Getenv(vaultApi.EnvVaultAddress), "Specify the Vault address")
	rootCmd.PersistentFlags().StringVar(&flags.Token, "vault-token", os.Getenv(vaultApi.EnvVaultToken), "Specify the Vault token")

	rootCmd.PersistentFlags().StringVarP(&flags.User, "user", "u", os.Getenv(EnvSSHUsername), "Your SSH username for templated configs")

	rootCmd.PersistentFlags().BoolVarP(&flags.StoredToken, "stored-token", "", false, "Use a stored token from 'vault login' (overrides --vault-token, auto-enabled when no token is specified)")
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Provide addition output")

	rootCmd.Flags().BoolVarP(&flags.Version, "version", "V", false, "Show the version")
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
