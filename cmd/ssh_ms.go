package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/cezmunsta/ssh_ms/ssh"
	"github.com/cezmunsta/ssh_ms/vault"

	"github.com/hashicorp/vault/api"
	credToken "github.com/hashicorp/vault/builtin/credential/token"
	login "github.com/hashicorp/vault/command"
	"github.com/spf13/cobra"
)

// EnvBasePath is the parent location used to prefix storage paths
const EnvBasePath = "HOME"

type cmdFlags struct {
	Expert, List, Purge, StoredToken, Simulate, Verbose, Version bool
	Addr, Show, StoragePath, Token, User, Write                  string
}

type secretData map[string]interface{}

type userInfo struct {
	Username     string
	Firstname    string
	Lastname     string
	Firstinitial string
	Lastinitial  string
}

var (
	rootCmd = &cobra.Command{
		Use:   "ssh_ms",
		Short: "ssh_ms connects you to customers",
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
			} else if flags.Expert {
				connectCmd.Run(cmd, args)
			} else {
				cmd.Usage()
				os.Exit(1)
			}
		},
	}

	connectCmd = &cobra.Command{
		Use:   "connect CONNECTION [flags]",
		Short: "Connect via SSH",
		Long:  "Connect to a chosen connection using a stored configuration",
		Example: `
    ssh_ms connect localhost
    ssh_ms connect localhost hostname
		`,
		Run: func(cmd *cobra.Command, args []string) {
			connect(*getVaultClient(), ssh.UserEnv{User: sshArgs.User, Simulate: flags.Simulate}, args)
		},
	}

	deleteCmd = &cobra.Command{
		Use:   "delete KEY [flags]",
		Short: "Delete a config",
		Long:  "Delete an existing SSH configuration from Vault",
		Example: `
    ssh_ms delete localhost 
		`,
		Run: func(cmd *cobra.Command, args []string) {
			deleteSecret(*getVaultClient(), args[0])
		},
	}

	listCmd = &cobra.Command{
		Use:   "list [flags]",
		Short: "List available connections",
		Long:  "Checks Vault to find available connections and lists them",
		Run: func(cmd *cobra.Command, args []string) {
			listSecrets(*getVaultClient())
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
			purgeCache()
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
			showSecret(*getVaultClient(), args[0])
		},
	}

	writeCmd = &cobra.Command{
		Use:   "write KEY [flags]",
		Short: "Write, or update a config",
		Long:  "Write a new SSH configuration to Vault, or replace an existing one",
		Example: `
    ssh_ms write localhost HostName=localhost Port=22 User=ceri
		`,
		Run: func(cmd *cobra.Command, args []string) {
			writeSecret(*getVaultClient(), args[0], args[1:])
		},
	}

	flags        = cmdFlags{}
	placeholders = map[string]string{
		"@@USER_INITIAL_LASTNAME":  "{{.Firstinitial}}{{.Lastname}}",
		"@@USER_LASTNAME_INITIAL":  "{{.Lastname}}{{.Firstinitial}}",
		"@@USER_FIRSTNAME_INITIAL": "{{.Firstname}}{{.Lastinitial}}",
		"@@USER_FIRSTNAME":         "{{.Firstname}}",
		"@@" + EnvSSHUsername:      "{{.Username}}",
	}
	sshArgs = ssh.Connection{}

	// Version can be set with `-ldflags "-X github.com/cezmunsta/ssh_ms/cmd.Version=xxx"`
	Version = "1.0"

	// EnvSSHUsername is used to authenticate with SSH
	EnvSSHUsername = "SSH_MS_USERNAME"

	// EnvSSHIdentityFile is used for SSH authentication
	EnvSSHIdentityFile = "id_rsa"

	// EnvVaultAddr is the default location for Vault
	EnvVaultAddr = api.EnvVaultAddress
)

// authenticate a user with Vault
// e : user environment
func authenticate(e vault.UserEnv) *api.Client {
	os.Setenv(api.EnvVaultAddress, e.Addr)
	defer os.Setenv(api.EnvVaultAddress, "")

	if !flags.StoredToken {
		os.Setenv(api.EnvVaultToken, e.Token)
	}
	defer os.Setenv(api.EnvVaultToken, "")

	os.Setenv(api.EnvVaultMaxRetries, "3")
	defer os.Setenv(api.EnvVaultMaxRetries, "")

	config := api.DefaultConfig()
	client, err := api.NewClient(config)

	if flags.StoredToken {
		lc := &login.LoginCommand{
			BaseCommand: &login.BaseCommand{},
			Handlers: map[string]login.LoginHandler{
				"token": &credToken.CLIHandler{},
			},
		}
		th, err := lc.TokenHelper()
		if err != nil {
			panic("Uh oh")
		}
		storedToken, err := th.Get()
		if err != nil {
			panic("Unable to read token from store")
		}
		client.SetToken(storedToken)
	}

	if err != nil {
		log.Println(api.EnvVaultAddress, e.Addr)
		log.Println("Client address", client.Address())
		log.Fatal(err)
	}

	client.Auth()
	return client
}

// prepareConnection for SSH
// vc : Vault client
// args : options for inspection
// verbose : enable informational output
func prepareConnection(vc api.Client, args []string, verbose bool) ([]string, ssh.Connection) {
	var sshArgs []string
	var config map[string]interface{}

	host := args[0]
	config = getLocalCache(host)

	if config == nil {
		config = getRemoteCache(vc, host)
	}

	if config == nil {
		return sshArgs, ssh.Connection{}
	}

	if verbose {
		log.Println("config:", config)
	}

	sshClient := ssh.Connection{}
	sshArgs = append(sshClient.BuildConnection(config, host), args[1:]...)

	if verbose {
		log.Println("SSH cmd:", sshArgs)
	}
	return sshArgs, sshClient
}

// getRemoteCache reads directly from Vault
// host : the hostname alias for SSH
func getRemoteCache(vc api.Client, host string) map[string]interface{} {
	var data map[string]interface{}
	secret := vault.ReadSecret(vc, fmt.Sprintf("secret/ssh_ms/%s", host))

	if secret != nil {
		data = secret.Data
		saveLocalCache(host, data)
	}

	return data
}

// saveLocalCache creates a local copy in JSON format
// host : the hostname alias for SSH
// data : the SSH configuration
func saveLocalCache(host string, data map[string]interface{}) error {
	buff, _ := json.Marshal(data)
	return ioutil.WriteFile(getLocalPath(host), []byte(string(buff)), 0644)
}

// getLocalPath provides the path to a host cache
// host : the hostname alias for SSH
func getLocalPath(host string) string {
	return filepath.Join(flags.StoragePath, host+".json")
}

// getLocal checks for a fresh, local copy
// host : SSH host alias
func getLocalCache(host string) map[string]interface{} {
	var data map[string]interface{}

	if err := os.MkdirAll(flags.StoragePath, os.ModePerm); err != nil {
		log.Println(err)
		return nil
	}

	if err := expireLocalCache(host); err != nil {
		log.Println(err)
		return nil
	}

	read, err := ioutil.ReadFile(getLocalPath(host))
	if err != nil {
		log.Println("No local copy exists for", host)
		return nil
	}

	json.Unmarshal(read, &data)
	return data
}

// expireLocalCache checks to see if the cache file is stale
// host : SSH host alias
func expireLocalCache(host string) error {
	cacheFile := getLocalPath(host)
	localCache, err := os.Stat(cacheFile)
	if err != nil {
		return err
	}

	loc, _ := time.LoadLocation("UTC")
	lastModified := localCache.ModTime().In(loc)
	expiresAt := lastModified.Add((7 * 24) * time.Hour)

	if time.Now().In(loc).After(expiresAt) {
		os.Remove(cacheFile)
	}
	return err
}

// purgeCache will remove all of the cached configurations
func purgeCache() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Purging: ", flags.StoragePath, " ... CTRL+C to abort")
	reader.ReadString('\n')

	if err := os.RemoveAll(flags.StoragePath); err != nil {
		log.Println("Problem purging cache:", err)
		return err
	}
	return nil
}

// connect using SSH
// addr : Vault address
// token : Vault token
// user : SSH user
// simulate : perform a dry-run
// list : list available hosts
// expert : enable full control over SSH args
// verbose : perform additional output
// args : extra args passed by the user
func connect(vc api.Client, env ssh.UserEnv, args []string) {
	sshArgs := args
	sshClient := ssh.Connection{}
	execCmd := true

	if flags.Verbose {
		log.Println("Env:", env)
		log.Println("Expert:", flags.Expert)
		log.Println("Args:", args)
	}

	if !flags.Expert {
		secrets := vault.ListSecrets(vc, "secret/ssh_ms")

		if secrets == nil || secrets.Data["keys"] == nil {
			log.Println("response: no secret")
		} else {
			sshArgs, sshClient = prepareConnection(vc, args, flags.Verbose)
		}
	}

	for i := 0; i < len(sshClient.LocalForward); i++ {
		log.Printf("FWD: %d -> %d\n", sshClient.LocalForward[i].LocalPort, sshClient.LocalForward[i].RemotePort)
	}

	if execCmd {
		ssh.Connect(sshArgs, env)
	}
}

// deleteSecret in Vault
// vc : Vault client
// key : secret to remove
func deleteSecret(vc api.Client, key string) bool {
	return vault.DeleteSecret(vc, fmt.Sprintf("secret/ssh_ms/%s", key))
}

// listSecrets in Vault
// vc : Vault client
func listSecrets(vc api.Client) bool {
	secrets := vault.ListSecrets(vc, "secret/ssh_ms")

	if secrets == nil || secrets.Data["keys"] == nil {
		log.Println("response: no secrets")
		return false
	}

	fmt.Println("available connections:")

	switch reflect.TypeOf(secrets.Data["keys"]).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(secrets.Data["keys"])

		for i := 0; i < s.Len(); i++ {
			m := " "
			if math.Mod(float64(i), 3) == 0 && i > 0 {
				m += "\n"
			}
			fmt.Print(s.Index(i), m)
		}
	}
	fmt.Println("")
	return true
}

// showSecret in Vault
// vc : Vault client
// key : secret to show
func showSecret(vc api.Client, key string) bool {
	secret := vault.ReadSecret(vc, fmt.Sprintf("secret/ssh_ms/%s", key))

	if secret == nil {
		log.Println("response: no secret")
		return false
	}

	sshClient := ssh.Connection{}
	sshArgs := sshClient.BuildConnection(secret.Data, key)
	config := rewriteEmpty(rewriteUsername(sshClient.Cache))

	fmt.Print(config)

	if flags.Verbose {
		log.Println("SSH cmd:", sshArgs)
	}

	return true
}

// writeSecret to Vault
// vc : Vault client
// key : secret key name
// args : extra args passed by the user
func writeSecret(vc api.Client, key string, args []string) bool {
	secret := make(secretData)
	status := true

	for i := 0; i < len(args); i++ {
		s := strings.Split(args[i], "=")
		secret[s[0]] = s[1]
	}

	if !flags.Simulate {
		status = vault.WriteSecret(vc, key, secret)
	} else {
		log.Println(" - secret would be: ", secret)
	}

	if flags.Verbose {
		log.Println("writeSecret status:", status)
	}

	return status
}

// rewriteUsername config templates
// cache : config connection details
func rewriteUsername(cache ssh.CachedConnection) string {
	var b bytes.Buffer

	updatedConfig := cache.Config
	name := strings.Split(flags.User, ".")

	if flags.Verbose {
		log.Println("rewriteUsername applying user", flags.User)
	}

	for marker, tpl := range placeholders {
		updatedConfig = strings.Replace(updatedConfig, marker, tpl, 1)
	}

	tmpl, err := template.New("dummy").Parse(updatedConfig)
	if err != nil {
		panic(err)
	}

	if len(name) > 0 && len(name[0]) > 0 {
		tmpl.Execute(&b, userInfo{strings.Join(name, "."), name[0], name[1], name[0][0:1], name[1][0:1]})
	} else {
		log.Println("*** Your LDAP username was undetected ****\n")
		tmpl.Execute(&b, userInfo{})
	}
	updatedConfig = b.String()

	return updatedConfig
}

// rewriteEmpty lines from a config
// config : string representation of the config
func rewriteEmpty(config string) string {
	re := regexp.MustCompile(`(\b(User|ProxyJump)\b)([[:space:]]+| none)?\n`)
	return re.ReplaceAllString(config, "\r")
}

// getVaultClient by authenticating
func getVaultClient() *api.Client {
	env := vault.UserEnv{Addr: flags.Addr, Token: flags.Token, Simulate: flags.Simulate}
	if flags.Verbose {
		log.Println("Vault Address:", env.Addr)
		log.Println("Simulate:", flags.Simulate)
	}
	return authenticate(env)
}

// printVersion of the application
func printVersion() {
	if !flags.Verbose {
		fmt.Println(Version)
	} else {
		fmt.Println("Version:", Version)
		fmt.Println("Arch:", runtime.GOOS, runtime.GOARCH)
	}
}

// Execute build all of the commands
func Execute() {
	rootCmd.AddCommand(
		connectCmd,
		deleteCmd,
		listCmd,
		purgeCmd,
		showCmd,
		writeCmd,
	)

	rootCmd.PersistentFlags().StringVar(&flags.Addr, "vault-addr", os.Getenv(api.EnvVaultAddress), "Specify the Vault address")
	rootCmd.PersistentFlags().StringVar(&flags.Token, "vault-token", os.Getenv(api.EnvVaultToken), "Specify the Vault token")
	rootCmd.PersistentFlags().StringVarP(&flags.User, "user", "u", os.Getenv(EnvSSHUsername), "Your SSH username for templated configs")
	rootCmd.PersistentFlags().StringVarP(&flags.StoragePath, "storage", "s", filepath.Join(os.Getenv(EnvBasePath), ".ssh", "cache"), "Storage path for caching")

	rootCmd.PersistentFlags().BoolVarP(&flags.StoredToken, "stored-token", "", false, "Use a stored token from 'vault login' (overrides --vault-token, auto-enabled when no token is specified)")
	rootCmd.PersistentFlags().BoolVarP(&flags.Simulate, "dry-run", "n", false, "Display commands rather than executing them")
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Provide addition output")

	rootCmd.Flags().BoolVarP(&flags.Version, "version", "V", false, "Show the version")

	connectCmd.Flags().BoolVarP(&flags.Expert, "expert", "e", false, "Expert-mode - pass in your SSH args")

	writeCmd.Flags().StringVarP(&sshArgs.HostName, "host", "H", sshArgs.HostName, "Set HostName")
	writeCmd.Flags().Uint16VarP(&sshArgs.Port, "port", "p", sshArgs.Port, "Set Port")
	writeCmd.Flags().StringVarP(&sshArgs.IdentityFile, "identity", "i", "~/.ssh/"+EnvSSHIdentityFile, "Set IdentityFile")
	writeCmd.Flags().StringVarP(&sshArgs.ProxyJump, "proxy", "P", sshArgs.ProxyJump, "Set ProxyJump")

	if flags.Token == "" {
		flags.StoredToken = true
	}
	if flags.Addr == "" {
		flags.Addr = EnvVaultAddr
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
