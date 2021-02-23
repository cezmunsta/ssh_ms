package cmd

import (
	"fmt"
	"os"
	"runtime"

	vaultApi "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
)

// EnvBasePath is the parent location used to prefix storage paths
const EnvBasePath = "HOME"

type cmdFlags struct {
	List, Verbose, Version bool
	Show                   string
}

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

	flags = cmdFlags{}

	// Version can be set with `-ldflags "-X github.com/cezmunsta/ssh_ms/cmd.Version=xxx"`
	Version = "1.0"

	// EnvSSHUsername is used to authenticate with SSH
	EnvSSHUsername = "SSH_MS_USERNAME"

	// EnvSSHIdentityFile is used for SSH authentication
	EnvSSHIdentityFile = "id_rsa"

	// EnvVaultAddr is the default location for Vault
	EnvVaultAddr = vaultApi.EnvVaultAddress
)

func init() {
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
		fmt.Println(line)
	}
}

// Execute the commands
func Execute() (int, error) {
	if err := rootCmd.Execute(); err != nil {
		return 1, err
	}
	return 0, nil
}
