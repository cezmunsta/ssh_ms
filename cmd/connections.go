package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	vaultApi "github.com/hashicorp/vault/api"

	"github.com/cezmunsta/ssh_ms/log"
	"github.com/cezmunsta/ssh_ms/ssh"
	vaultHelper "github.com/cezmunsta/ssh_ms/vault"
)

type secretData map[string]interface{}

// CacheExpireAfter sets the threshold for cleaning stales caches
const CacheExpireAfter = (7 * 24) * time.Hour

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
		log.Infof("simulated write to '%v': %v", key, args)
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

	if cfg.Simulate {
		log.Infof("simulated delete of '%v'", key)
		return true
	}

	status, err := vaultHelper.DeleteSecret(vc, fmt.Sprintf("%s/%s", SecretPath, key))
	if err != nil {
		log.Warning("Unable to delete connection", key)
		return false
	}
	return status
}

// prepareConnection for SSH
// vc : Vault client
// args : options for inspection
// verbose : enable informational output
func prepareConnection(vc *vaultApi.Client, args []string) ([]string, ssh.Connection) {
	log.Debugf("prepareConnection: %v", args)

	var sshArgs []string
	var config map[string]interface{}

	if len(args) == 0 {
		log.Fatal("Minimum requirement is to specify an alias")
	}
	key := args[0]
	config, _ = getCache(key)

	if config == nil {
		config, _ = getRemoteCache(vc, key)
	}

	if config == nil {
		return sshArgs, ssh.Connection{}
	}

	log.Debugf("config: %v", config)
	sshClient := ssh.Connection{}
	sshArgs = append(sshClient.BuildConnection(config, key, cfg.User), args[1:]...)
	log.Debugf("sshArgs: %v", sshArgs)
	return sshArgs, sshClient
}

// connect using SSH
// vc: Vault client
// env: UserEnv configuration
// args : extra args passed by the user
func connect(vc *vaultApi.Client, env ssh.UserEnv, args []string) {
	log.Debug("connect: %v", args[0])
	sshArgs, sshClient := prepareConnection(vc, args)

	log.Debugf("%v", map[string]interface{}{
		"env": env,
		//"expertMode": cfg.ExpertMode
		"args": args,
	})

	for i := 0; i < len(sshClient.LocalForward); i++ {
		fmt.Printf("FWD: https://127.0.0.1:%d -> %d\n", sshClient.LocalForward[i].LocalPort, sshClient.LocalForward[i].RemotePort)
	}

	if cfg.Simulate {
		printConnection(vc, args[0])
		return
	}
	ssh.Connect(sshArgs, env)
}

// getCachePath returns the path to save to
func getCachePath(key string) string {
	log.Debugf("getCachePath: %v", key)
	return filepath.Join(cfg.StoragePath, key+".json")
}

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

	/*if err := expireLocalCache(host); err != nil {
		log.Println(err)
		return nil
	}*/

	read, err := ioutil.ReadFile(getCachePath(key))
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

	if status, err := saveCache(key, conn.Data); err != nil || status == false {
		return nil, err
	}
	return conn.Data, nil
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

// purgeCache will remove the cache directory and its contents
func purgeCache() (bool, error) {
	log.Debugf("purgeCache")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Purging: ", cfg.StoragePath, " ... CTRL+C to abort")
	reader.ReadString('\n')

	if err := os.RemoveAll(cfg.StoragePath); err != nil {
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

	if err := ioutil.WriteFile(getCachePath(key), []byte(string(buff)), 0640); err != nil {
		log.Errorf("Failed to save cache for '%v': %v", key, err)
		return false, err
	}
	return true, nil
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
