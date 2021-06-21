package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/cezmunsta/ssh_ms/config"
	"github.com/cezmunsta/ssh_ms/log"
	"github.com/cezmunsta/ssh_ms/ssh"
	vaultApi "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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

	connectCmd = &cobra.Command{
		Use:   "connect CONNECTION",
		Short: "Connect to a host",
		Long:  "Connect to a host using the stored configuration",
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(args, 1)
			connect(getVaultClient(), ssh.UserEnv{User: cfg.User, Simulate: cfg.Simulate}, args)
		},
	}

	deleteCmd = &cobra.Command{
		Use:   "delete CONNECTION [flags]",
		Short: "Delete a connection",
		Long:  "Lookup the requested connection and remove it when present",
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(args, 1)
			deleteConnection(getVaultClient(), args[0])
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
		Use:   "print CONNECTION [flags]",
		Short: "Print out the SSH command for a connection",
		Long:  "Print full command that would be used to connect",
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(args, 1)
			printConnection(getVaultClient(), args[0])
		},
	}

	purgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Purge the cache",
		Long:  "Remove all of the cached configurations",
		Example: `
	ssh_ms purge
        `,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: add logic to allow selective purge
			purgeCache()
		},
	}

	searchCmd = &cobra.Command{
		Use:   "search PATTERN [flags]",
		Short: "Search for a connection",
		Long:  "Search the list of connections using a pattern",
		Example: `
	ssh_ms search gate
	ssh_ms search '^g.*'
	ssh_ms search 'way$'
        `,
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(args, 1)
			searchConnections(getVaultClient(), args[0])
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
			checkArgs(args, 1)
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

	updateCmd = &cobra.Command{
		Use:   "update CONNECTION [args]",
		Short: "Update an existing connection to storage",
		Long:  "TBD",
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(args, 1)
			updateConnection(getVaultClient(), args[0], args[1:])
		},
	}

	writeCmd = &cobra.Command{
		Use:   "write CONNECTION [args]",
		Short: "Add a new connection to storage",
		Long:  "TBD",
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(args, 1)
			writeConnection(getVaultClient(), args[0], args[1:])
		},
	}

	cfg = config.GetConfig()

	/*
		The following support overrides during builds, which can be done
		by setting ldflags, e.g.

		`-ldflags "-X github.com/cezmunsta/ssh_ms/cmd.Version=xxx"`

	*/

	// Version of the code
	Version = "1.2.2"
)

func init() {
	rootCmd.AddCommand(
		connectCmd,
		deleteCmd,
		listCmd,
		printCmd,
		purgeCmd,
		searchCmd,
		showCmd,
		versionCmd,
		updateCmd,
		writeCmd,
	)
	rootCmd.PersistentFlags().StringVar(&cfg.VaultAddr, "vault-addr", cfg.EnvVaultAddr, "Specify the Vault address")
	rootCmd.PersistentFlags().StringVar(&cfg.VaultToken, "vault-token", os.Getenv(vaultApi.EnvVaultToken), "Specify the Vault token")

	rootCmd.PersistentFlags().StringVarP(&cfg.StoragePath, "storage", "s", cfg.StoragePath, "Storage path for caching")
	rootCmd.PersistentFlags().StringVarP(&cfg.User, "user", "u", os.Getenv(cfg.EnvSSHUsername), "Your SSH username for templated configs")

	rootCmd.PersistentFlags().BoolVarP(&cfg.StoredToken, "stored-token", "", false, "Use a stored token from 'vault login' (overrides --vault-token, auto-enabled when no token is specified)")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Debug, "debug", "d", false, "Provide addition output")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Simulate, "dry-run", "n", false, "Prevent certain commands without full execution")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Provide addition output")

	updateCmd.Flags().StringVarP(&cfg.ConfigComment, "comment", "c", "", "Set the comment for the config entry")
	writeCmd.Flags().StringVarP(&cfg.ConfigComment, "comment", "c", "", "Add a comment for the config entry")
	updateCmd.Flags().StringVarP(&cfg.ConfigMotd, "motd", "m", "", "Set the Motd for the config entry")
	writeCmd.Flags().StringVarP(&cfg.ConfigMotd, "motd", "m", "", "Add a Motd comment for the config entry")

	log := log.GetLogger(log.GetDefaultLevel(), "")
	cfg.LogLevel = log.GetLevel()
}

// checkArgs makes sure that at least a certain number of args exist
func checkArgs(args []string, min int) {
	if len(args) < min {
		log.Fatal("Missing argument for connection")
	}
}

// getVersion information for the application
func getVersion() [][]string {
	var lines [][]string

	if !cfg.Verbose && !cfg.Debug {
		lines = append(lines, []string{Version})
	} else {
		lines = append(lines, []string{"Version:", Version})
		lines = append(lines, []string{"Arch:", runtime.GOOS, runtime.GOARCH})
		lines = append(lines, []string{"Base path:", config.EnvBasePath})
		lines = append(lines, []string{"Default Vault address:", config.EnvVaultAddr})
		lines = append(lines, []string{"Default SSH username:", config.EnvSSHDefaultUsername})
		lines = append(lines, []string{"SSH template username:", config.EnvSSHUsername})
		lines = append(lines, []string{"SSH identity file:", config.EnvSSHIdentityFile})
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

	log.Debug("config: ", cfg.ToJSON())
}

// Execute the commands
func Execute() (int, error) {
	if err := rootCmd.Execute(); err != nil {
		return 1, err
	}
	return 0, nil
}
