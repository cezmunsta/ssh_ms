package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/cezmunsta/ssh_ms/config"
	"github.com/cezmunsta/ssh_ms/log"
	vaultApi "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// EnvBasePath is the parent location used to prefix storage paths
const EnvBasePath = "HOME"

var (
	rootCmd = &cobra.Command{
		Use:   "ssh_ms",
		Short: "ssh_ms connects you to your remote world",
		Long:  "ssh_ms integrates with HashiCorp Vault to store SSH configs and ease your remote life",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Name() == "help" {
				return
			}
			updateSettings()
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
			os.Exit(1)
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

	printCmd = &cobra.Command{
		Use:   "print [flags]",
		Short: "Print out the SSH command for a connection",
		Long:  "Print full command that would be used to connect",
		Run: func(cmd *cobra.Command, args []string) {
			printConnection(getVaultClient(), args[0])
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

	versionCmd = &cobra.Command{
		Use:   "version [flags]",
		Short: "Show the version",
		Long:  "Show the version of the application",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
	}

	cfg = config.GetConfig()

	/*
		The following support overrides during builds, which can be done
		by setting ldflags, e.g.

		`-ldflags "-X github.com/cezmunsta/ssh_ms/cmd.Version=xxx"`

	*/

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
		printCmd,
		showCmd,
		versionCmd,
	)
	rootCmd.PersistentFlags().StringVar(&cfg.VaultAddr, "vault-addr", os.Getenv(vaultApi.EnvVaultAddress), "Specify the Vault address")
	rootCmd.PersistentFlags().StringVar(&cfg.VaultToken, "vault-token", os.Getenv(vaultApi.EnvVaultToken), "Specify the Vault token")

	rootCmd.PersistentFlags().StringVarP(&cfg.User, "user", "u", os.Getenv(cfg.EnvSSHUsername), "Your SSH username for templated configs")

	rootCmd.PersistentFlags().BoolVarP(&cfg.StoredToken, "stored-token", "", false, "Use a stored token from 'vault login' (overrides --vault-token, auto-enabled when no token is specified)")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Debug, "debug", "d", false, "Provide addition output")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Provide addition output")

	log := log.GetLogger(log.GetDefaultLevel(), "")
	cfg.LogLevel = log.GetLevel()
}

// getVersion information for the application
func getVersion() [][]string {
	var lines [][]string

	if !cfg.Verbose && !cfg.Debug {
		lines = append(lines, []string{Version})
	} else {
		lines = append(lines, []string{"Version:", Version})
		lines = append(lines, []string{"Arch:", runtime.GOOS, runtime.GOARCH})
	}
	return lines
}

// printVersion of the application
func printVersion() {
	for _, line := range getVersion() {
		fmt.Println(strings.Join(line, " "))
	}
}

// updateSettings will update certain configuration items
func updateSettings() {
	if cfg.Debug {
		cfg.LogLevel = logrus.DebugLevel
	} else if cfg.Verbose {
		cfg.LogLevel = logrus.InfoLevel
	}
	log.SetLevel(cfg.LogLevel)

	if cfg.VaultToken == "" {
		cfg.StoredToken = true
	}

	if cfg.VaultAddr == "" {
		cfg.VaultAddr = EnvVaultAddr
	}
}

// Execute the commands
func Execute() (int, error) {
	if err := rootCmd.Execute(); err != nil {
		return 1, err
	}
	return 0, nil
}
