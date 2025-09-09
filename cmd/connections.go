package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	vaultApi "github.com/hashicorp/vault/api"

	"github.com/cezmunsta/ssh_ms/log"
	"github.com/cezmunsta/ssh_ms/ssh"
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
)

type secretData map[string]interface{}

type configMotdTpl struct {
	Comment, Motd, Name string
}

var currentCommand string

const (
	// CacheExpireAfter sets the threshold for cleaning stale caches
	CacheExpireAfter = (7 * 24) * time.Hour

	// LockPrefix is used to manage locking
	LockPrefix = "ssh_ms_lock_"
)

// getVaultClient by authenticating using flags
func getVaultClient() *vaultApi.Client {
	env := vaultHelper.UserEnv{
		Addr:     cfg.VaultAddr,
		Token:    cfg.VaultToken,
		Simulate: cfg.Simulate,
	}
	return getVaultClientWithEnv(env)
}

// getVaultClientWithEnv by authenticating using UserEnv
func getVaultClientWithEnv(env vaultHelper.UserEnv) *vaultApi.Client {
	if cfg.Verbose {
		log.Debug("Vault Address:", env.Addr)
		log.Debug("Simulate:", cfg.Simulate)
	}
	return vaultHelper.Authenticate(env, cfg.StoredToken)
}

// getLockName produces a lock path
func getLockName(key string) string {
	return fmt.Sprintf("%s_%s", LockPrefix, key)
}

// getLockPath produces the full lock path
func getLockPath(ln string) string {
	return fmt.Sprintf("%s/%s", getSecretPath(), ln)
}

// getSecretPath produces the secret path
func getSecretPath() string {
	sp := strings.Split(cfg.SecretPath, ",")
	p := sp[0]

	if len(sp) > 1 && cfg.NameSpace == "" {
		log.Warningf("multiple namespaces exist (%v), setting --namespace to the default '%s'", cfg.SecretPath, sp[0])
		cfg.NameSpace = sp[0]
	}

	for _, ap := range sp {
		if ap == cfg.NameSpace {
			p = ap
			break
		}
	}

	return p
}

// acquireLock creates a lock to control writes
func acquireLock(vc *vaultApi.Client, key string) (bool, string) {
	log.Debug("acquireLock: ", key)
	loc, _ := time.LoadLocation("UTC")
	ln := getLockName(key)

	conn := make(secretData)
	conn["User"] = os.Getenv(cfg.EnvSSHUsername)

	// TODO: Auto-handle expired locks if necessary
	/*
		if existingLock, err := getRawConnection(vc, key); err != nil || existingLock != nil {
			loc, _ := time.LoadLocation("UTC")
			if val, ok := existingLock.Data["Expires"]; ok {
				et, _ := time.ParseInLocation(time.RFC3339, fmt.Sprintf("%v", val), loc)
				if time.Now().In(loc).After(et) {
					vault.DeleteSecret()
				} else {

				}
			} else {
				log.Panicf("Unexpected situation, lock appears to be missing expiry date: ", existingLock.Data)
			}
		}
	*/

	if existingLock, err := getRawConnection(vc, ln); existingLock != nil {
		log.Warningf("The record for '%v' appears to be locked: %v", key, existingLock)
		if err != nil {
			log.Fatal("existingLock error:", err)
		}
		return false, "nolock"
	}

	conn["Expires"] = time.Now().In(loc).Add(10 * time.Minute)

	status, err := vaultHelper.WriteSecret(vc, getLockPath(ln), conn)
	if err != nil {
		log.Fatalf("Failed to acquire lock for '%v': %v", key, err)
		return false, "nolock"
	}
	return status, ln
}

// releaseLock will remove the acquired lock
func releaseLock(vc *vaultApi.Client, ln string) (bool, error) {
	log.Debug("releaseLock:", ln)
	existingLock, err := getRawConnection(vc, ln)
	if err != nil {
		log.Error("existingLock error:", err)
		return false, err
	}

	if existingLock == nil {
		log.Debug("Unable to find lock for:", ln)
		return false, nil
	}

	status, err := vaultHelper.DeleteSecret(vc, getLockPath(ln))
	if err != nil {
		log.Errorf("Failed to release lock for '%v': %v", ln, err)
		return false, err
	}
	return status, nil
}

// getConnections from Vault
func getConnections(vc *vaultApi.Client) ([]string, error) {
	var connections []string

	sp := cfg.SecretPath
	if (currentCommand != "list" && currentCommand != "search") || cfg.NameSpace != "" {
		sp = getSecretPath()
	}

	secrets, err := vaultHelper.ListSecrets(vc, sp)

	if err != nil && len(secrets) == 0 {
		log.Fatalf("Unable to get connections for %v: %v", vc.Address(), err)
	} else if len(secrets) == 0 {
		return nil, errors.New("no data returned")
	}

	for _, secret := range secrets {
		if secret == nil || secret.Data["keys"] == nil {
			continue
		}

		switch reflect.TypeOf(secret.Data["keys"]).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(secret.Data["keys"])
			for i := 0; i < s.Len(); i++ {
				connections = append(connections, fmt.Sprintf("%s", s.Index(i)))
			}
		}
	}
	return connections, nil
}

// listConnections from Vault
func listConnections(vc *vaultApi.Client) bool {
	log.Debugf("listConnections")
	currentCommand = "list"
	return searchConnections(vc, ".*")
}

// searchConnections filters the list of connections
func searchConnections(vc *vaultApi.Client, pattern string) bool {
	log.Debug("searchConnections: ", pattern)
	currentCommand = "search"
	connections, err := getConnections(vc)
	search := regexp.MustCompile(pattern)
	ignore := regexp.MustCompile("^" + LockPrefix + ".*")
	c := 0

	if connections == nil || err != nil {
		fmt.Println("no available connections")
		return false
	}

	for _, s := range connections {
		m := " "
		if (pattern != ".*" && !search.MatchString(s)) || ignore.MatchString(s) {
			continue
		}
		c++
		if math.Mod(float64(c), 3) == 0 && c > 0 {
			m += "\n"
		}
		fmt.Print(s, m)
	}
	fmt.Println("")
	return true
}

// showConnection details suitable for use with ssh_config
func showConnection(vc *vaultApi.Client, key string) bool {
	log.Debugf("showConnection: %v", key)
	currentCommand = "show"
	sshArgs, sshClient, configComment, _ := prepareConnection(vc, []string{key})

	log.Info("SSH cmd:", sshArgs)
	if len(configComment) > 0 {
		fmt.Println("#", configComment)
	} else {
		fmt.Println("#", key)
	}
	fmt.Println(sshClient.Cache.Config)
	return true
}

// printConnection details suitable for use on the command line
func printConnection(vc *vaultApi.Client, key string) bool {
	log.Debugf("printConnection: %v", key)
	currentCommand = "print"
	sshArgs, _, _, _ := prepareConnection(vc, []string{key})

	fmt.Printf("ssh %v\n", strings.Join(sshArgs, " "))
	return true
}

// writeConnection creates a new entry, or updates an existing one
func writeConnection(vc *vaultApi.Client, key string, args []string) bool {
	log.Debugf("writeConnection: %v", key)
	currentCommand = "write"
	_, err := getRawConnection(vc, key)
	conn := make(secretData)

	if err != nil {
		// New connection
		for i := 0; i < len(args); i++ {
			s := strings.Split(args[i], "=")
			conn[s[0]] = s[1]
		}
	} else {
		// Existing connection
		log.Warningf("Existing connection found for '%v', please use update instead", key)
		return false
	}

	conn["ConfigComment"] = cfg.ConfigComment
	conn["ConfigMotd"] = cfg.ConfigMotd

	if cfg.Simulate {
		log.Infof("simulated write to '%v': %v", key, args)
		return true
	}

	if status, lockName := acquireLock(vc, key); status && lockName != "nolock" {
		defer releaseLock(vc, lockName)
	} else {
		log.Fatal("Failed to acquire lock for writeConnection")
		return false
	}

	status, err := vaultHelper.WriteSecret(vc, fmt.Sprintf("%s/%s", getSecretPath(), key), conn)
	if err != nil {
		log.Errorf("Failed to write '%v': %v", key, err)
		return false
	}
	saveCache(key, conn)
	return status
}

// updateConnection performs a partial update of an existing connection
func updateConnection(vc *vaultApi.Client, key string, args []string) bool {
	log.Debugf("updateConnection: %v", key)
	currentCommand = "update"
	conn, err := getRawConnection(vc, key)
	if err != nil {
		log.Warningf("Unable to retrieve connection '%v', please use write instead", key)
		return false
	}

	for i := 0; i < len(args); i++ {
		s := strings.Split(args[i], "=")
		if len(s) != 2 {
			log.Fatalf("Unexpected option '%v', expected XXX=YYY", args[i])
		}
		conn[s[0]] = s[1]
	}

	if len(cfg.ConfigComment) > 0 {
		conn["ConfigComment"] = cfg.ConfigComment
	}

	if len(cfg.ConfigMotd) > 0 {
		conn["ConfigMotd"] = cfg.ConfigMotd
	}

	if cfg.Simulate {
		log.Infof("Simulate update of '%v': %v", key, conn)
		return true
	}

	if status, lockName := acquireLock(vc, key); status && lockName != "nolock" {
		defer releaseLock(vc, lockName)
	} else {
		log.Debugf("status: %v, lockName: %v", status, lockName)
		log.Fatal("Failed to acquire lock for updateConnection")
		return false
	}

	status, err := vaultHelper.WriteSecret(vc, fmt.Sprintf("%s/%s", getSecretPath(), key), conn)
	if err != nil {
		log.Errorf("Failed to write '%v': %v", key, err)
		return false
	}
	saveCache(key, conn)
	return status
}

// deleteConnection removes an entry from Vault
func deleteConnection(vc *vaultApi.Client, key string) bool {
	log.Debugf("deleteConnection: %v", key)
	currentCommand = "delete"
	_, err := getRawConnection(vc, key)
	if err != nil {
		log.Debug("Unable to retrieve connection", key)
		return false
	}
	if cfg.Simulate {
		log.Infof("simulated delete of '%v'", key)
		return true
	}

	if status, lockName := acquireLock(vc, key); status && lockName != "nolock" {
		defer releaseLock(vc, lockName)
	} else {
		log.Fatal("Failed to acquire lock for deleteConnection")
		return false
	}

	status, err := vaultHelper.DeleteSecret(vc, fmt.Sprintf("%s/%s", getSecretPath(), key))
	if err != nil {
		log.Warning("Unable to delete connection", key)
		return false
	}
	return status
}

func findKeyByValue(m map[string]string, value string) (string, bool) {
	for k, v := range m {
		if v == value {
			return k, true
		}
	}
	return "", false
}

// prepareConnection for SSH
// vc : Vault client
// args : options for inspection
// verbose : enable informational output
func prepareConnection(vc *vaultApi.Client, args []string) ([]string, ssh.Connection, string, string) {
	log.Debugf("prepareConnection: %v", args)
	var sshArgs []string
	var svc string
	var configComment string
	var configMotd string

	if len(args) == 0 {
		log.Fatal("Minimum requirement is to specify an alias")
	}
	key := args[0]
	config := lookupConnection(vc, key)
	configComment = key
	configMotd = ""

	if val, ok := config["ConfigComment"]; ok {
		log.Debugf("Found comment for '%v': %v", key, val)
		configComment = fmt.Sprintf("%v", val)
	}

	if val, ok := config["ConfigMotd"]; ok {
		log.Debugf("Found Motd for '%v': %v", key, val)
		configMotd = fmt.Sprintf("%v\n", val)
	}

	if config == nil {
		return sshArgs, ssh.Connection{}, key, configMotd
	}

	log.Debugf("config: %v", config)
	sshClient := ssh.Connection{}
	sshArgs = append(sshClient.BuildConnection(config, key, cfg.User), args[1:]...)
	log.Debugf("sshArgs: %v", sshArgs)

	for i := 0; i < len(sshClient.LocalForward); i++ {
		if svcName, ok := findKeyByValue(cfg.ServiceMap, strconv.FormatUint(uint64(sshClient.LocalForward[i].RemotePort), 10)); ok {
			svc = svcName
		} else {
			svc = "Custom forwarding"
		}
		configMotd += fmt.Sprintf("\nFWD: https://127.0.0.1:%d - %s (%d)", sshClient.LocalForward[i].LocalPort, svc, sshClient.LocalForward[i].RemotePort)
	}

	if configAutoMotdTpl, err := template.New("configAutoMotd").Parse(`
***************************************************************
# {{.Comment}}
Server connection: {{.Name}}

{{.Motd}}
***************************************************************

	`); err != nil {
		log.Warningf("Failed to proces MOTD for '%v': %v", key, err)
	} else {
		b := bytes.Buffer{}
		configAutoMotdTpl.Execute(&b, configMotdTpl{Comment: configComment, Motd: configMotd, Name: key})
		configMotd = b.String()
	}

	return sshArgs, sshClient, configComment, configMotd
}

// lookupConnection tries local cache and then remote to acquire connection details
func lookupConnection(vc *vaultApi.Client, key string) map[string]interface{} {
	log.Debug("lookupConnection: ", key)
	config, _ := getCache(key)

	if config == nil {
		config, _ = getRemoteCache(vc, key)
	}

	return config
}

// connect using SSH
// vc: Vault client
// env: UserEnv configuration
// args : extra args passed by the user
func connect(vc *vaultApi.Client, env ssh.UserEnv, args []string) {
	log.Debug("connect:", args[0])
	currentCommand = "connect"
	sshArgs, _, _, configMotd := prepareConnection(vc, args)

	log.Debugf("%v", map[string]interface{}{
		"env":  env,
		"args": args,
	})

	if cfg.Simulate {
		printConnection(vc, args[0])
		return
	}

	fmt.Println(configMotd)
	ssh.Connect(sshArgs, env)
}

// getCachePath returns the path to save to
func getCachePath(key string) string {
	log.Debugf("getCachePath: %v", key)
	return filepath.Join(cfg.StoragePath, key+".json")
}

// makeCachePath manages the creation of the cfg.StoragePath
func makeCachePath() (bool, error) {
	log.Debugf("makeCachePath: %v", cfg.StoragePath)
	if err := os.MkdirAll(cfg.StoragePath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create cache directory '%v': %v", cfg.StoragePath, err)
		return false, err
	}
	return true, nil
}

// getCache checks for a fresh, local copy
// key: SSH host alias
func getCache(key string) (map[string]interface{}, error) {
	log.Debugf("getCache: %v", key)
	var data map[string]interface{}
	makeCachePath()

	if _, err := expireCache(key); err != nil {
		log.Info("Failed to expireCache: ", err)
		return nil, err
	}

	read, err := os.ReadFile(getCachePath(key))
	if err != nil {
		log.Infof("No local copy exists for: %v", key)
		return nil, err
	}

	if err := json.Unmarshal(read, &data); err != nil {
		log.Fatalf("Failed to unmarshal data for '%v': %v", key, err)
	}
	return data, nil
}

// getRemoteCache reads directly from Vault
// key: the hostname alias for SSH
func getRemoteCache(vc *vaultApi.Client, key string) (map[string]interface{}, error) {
	log.Debugf("getRemoteCache: %v", key)
	conn, err := getRawConnection(vc, key)
	if err != nil {
		log.Errorf("Failed to request data for '%v': %v", key, err)
		return nil, err
	}

	if status, err := saveCache(key, conn); err != nil || !status {
		return nil, err
	}
	return conn, nil
}

// removeCache deletes a specific file
func removeCache(key string) (bool, error) {
	log.Debugf("removeCache: %v", key)
	cacheFile := getCachePath(key)

	if err := os.Remove(cacheFile); err != nil {
		if os.IsExist(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// expireLocalCache checks to see if the cache file is stale
// key: SSH host alias
func expireCache(key string) (bool, error) {
	log.Debugf("expireCache: %v", key)
	cacheFile := getCachePath(key)

	localCache, err := os.Stat(cacheFile)
	if err != nil {
		return false, err
	}

	loc, _ := time.LoadLocation("UTC")
	lastModified := localCache.ModTime().In(loc)

	if time.Now().In(loc).After(lastModified.Add(CacheExpireAfter)) {
		return removeCache(key)
	}
	return false, nil
}

// populateCache will create cache items for each of the available connections
func populateCache(vc *vaultApi.Client) (bool, error) {
	totalConnections := 0
	totalErrors := 0
	if connections, err := getConnections(vc); err == nil {
		totalConnections = len(connections)
		for _, conn := range connections {
			if localCache, _ := getCache(conn); localCache != nil {
				// Ignore any already-cached connections
				continue
			}
			if rawConn, err := getRawConnection(vc, conn); err != nil {
				log.Error("failed to get connection", conn)
				totalErrors++
				continue
			} else if _, err := saveCache(conn, rawConn); err != nil {
				log.Errorf("failed to save connection %s : %v", conn, err)
				totalErrors++
				continue
			}
			log.Debug("populated cache for", conn)
		}

	}
	if totalErrors == 0 {
		return true, nil
	}
	return false, fmt.Errorf("failed to populate cache, failure rate %v%%", 100*(totalErrors/totalConnections))
}

// purgeCache will remove the cache directory and its contents, or an individual item's cache if requested
func purgeCache() (bool, error) {
	log.Debugf("purgeCache")
	targeted := purgeConnection != "" && purgeConnection != "all"
	currentCommand = "purge"

	if !purgeForce {
		reader := bufio.NewReader(os.Stdin)
		if targeted {
			fmt.Print("Purging cached item: ", cfg.StoragePath+"/"+purgeConnection, " ... CTRL+C to abort")
		} else {
			fmt.Print("Purging entire cache: ", cfg.StoragePath, " ... CTRL+C to abort")
		}
		reader.ReadString('\n')
	}

	if targeted {
		log.Debug("Purging individual connection:", purgeConnection)
		return removeCache(purgeConnection)
	} else if err := os.RemoveAll(cfg.StoragePath); err != nil {
		log.Errorf("Problem purging cache: %v", err)
		return false, err
	}
	return true, nil
}

// saveCache creates a local copy in JSON format
// key: the hostname alias for SSH
// data : the SSH configuration
func saveCache(key string, data map[string]interface{}) (bool, error) {
	log.Debugf("saveCache: %v", key)
	makeCachePath()

	if err := os.MkdirAll(cfg.StoragePath, os.ModePerm); err != nil {
		log.Errorf("Failed to create cache directory '%v': %v", cfg.StoragePath, err)
		return false, err
	}

	buff, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Failed to generate JSON to cache '%v': %v", key, err)
		return false, err
	}

	if err := os.WriteFile(getCachePath(key), []byte(string(buff)), 0o640); err != nil {
		log.Errorf("Failed to save cache for '%v': %v", key, err)
		return false, err
	}
	return true, nil
}

// getRawConnection retrieves the secret from Vault
func getRawConnection(vc *vaultApi.Client, key string) (map[string]interface{}, error) {
	secret, err := vaultHelper.ReadSecret(vc, fmt.Sprintf("%s/%s", getSecretPath(), key))

	if err != nil || secret == nil {
		if !strings.HasPrefix(key, LockPrefix) && currentCommand != "write" {
			log.Warning("Unable to find connection for: ", key)
			return nil, errors.New("no match found")
		}
		return nil, errors.New("no lock found")
	}
	return secret, nil
}
