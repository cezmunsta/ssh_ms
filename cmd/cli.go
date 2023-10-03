package cmd

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

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

	cacheCmd = &cobra.Command{
		Use:   "cache",
		Short: "Cache management",
		Long:  "Manage your local cached connections stored in " + cfg.StoragePath,
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

	populateCacheCmd = &cobra.Command{
		Use:   "populate [flags]",
		Short: "Populate your local cache for all available connections",
		Long:  "To speed up your connections, or to assist avoiding issues when Vault might be unavailable, pre-populate your cache",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := populateCache(getVaultClient()); err != nil {
				log.Error(err)
			}
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

	purgeCacheCmd = &cobra.Command{
		Use:   "purge [flags]",
		Short: "Purge the cache",
		Long:  "Remove all of the cached configurations",
		Example: `
	ssh_ms purge
	ssh_ms purge -f
	ssh_ms purge -c conn1
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

	// Internals
	inspectCmd = &cobra.Command{
		Use:   "inspect ITEM",
		Short: "Inspect the value of an internal item",
		Long:  "TBD",
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(args, 1)
			inspectItem(args[0])
		},
	}

	cfg = config.GetConfig()

	/*
		The following support overrides during builds, which can be done
		by setting ldflags, e.g.

		`-ldflags "-X github.com/cezmunsta/ssh_ms/cmd.Version=xxx"`

	*/

	// Purge flags
	purgeConnection string
	purgeForce      bool

	// Version of the code
	Version = "1.10.2"
)

func init() {
	cacheCmd.AddCommand(
		populateCacheCmd,
		purgeCacheCmd,
	)
	rootCmd.AddCommand(
		cacheCmd,
		connectCmd,
		deleteCmd,
		inspectCmd,
		listCmd,
		printCmd,
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

	rootCmd.PersistentFlags().BoolVarP(&cfg.StoredToken, "stored-token", "", false,
		"Use a stored token from 'vault login' (overrides --vault-token, auto-enabled when no token is specified)")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Debug, "debug", "d", false, "Provide addition output")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Simulate, "dry-run", "n", false, "Prevent certain commands without full execution")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Provide addition output")

	connectCmd.Flags().StringVarP(&cfg.CustomLocalForward, "local-forward", "l", "",
		"Define adhoc LocalForward rules by specifying the target ports, e.g. -l 8080,3306")

	purgeCacheCmd.Flags().BoolVarP(&purgeForce, "force", "f", false, "Bypass confirmation prompt")
	purgeCacheCmd.Flags().StringVarP(&purgeConnection, "connection", "c", "", "Select a connection to purge")

	updateCmd.Flags().StringVarP(&cfg.ConfigComment, "comment", "c", "", "Set the comment for the config entry")
	writeCmd.Flags().StringVarP(&cfg.ConfigComment, "comment", "c", "", "Add a comment for the config entry")

	updateCmd.Flags().StringVarP(&cfg.ConfigMotd, "motd", "m", "", "Set the Motd for the config entry")
	writeCmd.Flags().StringVarP(&cfg.ConfigMotd, "motd", "m", "", "Add a Motd comment for the config entry")

	connectCmd.Flags().StringVarP(&cfg.NameSpace, "namespace", "N", "", "Specify the namespace for the config entry")
	deleteCmd.Flags().StringVarP(&cfg.NameSpace, "namespace", "N", "", "Specify the namespace for the config entry")
	listCmd.Flags().StringVarP(&cfg.NameSpace, "namespace", "N", "", "Specify the namespace for the config entry")
	showCmd.Flags().StringVarP(&cfg.NameSpace, "namespace", "N", "", "Specify the namespace for the config entry")
	updateCmd.Flags().StringVarP(&cfg.NameSpace, "namespace", "N", "", "Add a namespace for the config entry")
	writeCmd.Flags().StringVarP(&cfg.NameSpace, "namespace", "N", "", "Set the namespace for the config entry")

	versionCmd.Flags().BoolVarP(&cfg.VersionCheck, "check", "c", false, "Check for the latest version")

	log := log.GetLogger(log.GetDefaultLevel(), "")
	cfg.LogLevel = log.GetLevel()
}

// checkArgs makes sure that at least a certain number of args exist
func checkArgs(args []string, min int) {
	if len(args) < min {
		log.Fatal("Missing argument for connection")
	}
}

// checkVersion against the latest release
func checkVersion() [][]string {
	var lines [][]string
	var redirectURL []string

	url := "https://github.com/cezmunsta/ssh_ms/releases/latest"
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Cache-Control", "no-cache")
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectURL = req.Response.Header["Location"]
			return http.ErrUseLastResponse
		},
		Timeout: time.Second * 10,
	}

	if _, err := client.Do(req); err != nil {
		log.Debugf("Request: %v", req)
		log.Fatalf("Failed to lookup %s", url)
	}

	if parts := strings.Split(redirectURL[0], "/"); parts != nil {
		ver := strings.Replace(parts[len(parts)-1], "v", "", 1)

		if ver != Version {
			lines = append(lines, []string{"Latest Version:", ver})
			lines = append(lines, []string{"Download the latest version from", redirectURL[0]})
		} else {
			lines = append(lines, []string{"You are using the latest version"})
		}
	}

	return lines
}

// getVersion information for the application
func getVersion() [][]string {
	var lines [][]string

	if cfg.VersionCheck {
		lines = append(lines, checkVersion()...)
	}

	if !cfg.Verbose && !cfg.Debug {
		lines = append(lines, []string{Version})
	} else {
		lines = append(lines, []string{"Version:", Version})
		lines = append(lines, []string{"Arch:", runtime.GOOS, runtime.GOARCH})
		lines = append(lines, []string{"Go Version:", runtime.Version()})
		lines = append(lines, []string{"Vault Version:", cfg.VaultVersion})
		lines = append(lines, []string{"Base path:", config.EnvBasePath})
		lines = append(lines, []string{"Namespaces:\n-", strings.Join(strings.Split(config.SecretPath, ","), "\n- ")})
		lines = append(lines, []string{"Default Vault address:", config.EnvVaultAddr})
		lines = append(lines, []string{"Default SSH username:", config.EnvSSHDefaultUsername})
		lines = append(lines, []string{"SSH template username:", config.EnvSSHUsername})
		lines = append(lines, []string{"SSH identity file:", config.EnvSSHIdentityFile})
	}
	return lines
}

// inspectItem allows display the value of an item in the allow-list
func inspectItem(item string) {
	switch item {
	case "placeholders", "ph":
		for k, v := range ssh.Placeholders {
			if cfg.Verbose {
				fmt.Printf("%v = %v\n", k, v)
			} else {
				fmt.Println(k)
			}
		}
	}
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
