package ssh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/cezmunsta/ssh_ms/config"
	"github.com/cezmunsta/ssh_ms/log"
)

var (
	errNoFreePort       = fmt.Errorf("no free port")
	localForwardPortMin = uint16(18000)
	localForwardPortMax = uint16(20000)

	cfg = config.GetConfig()

	// Placeholders are used for templated connections
	Placeholders = map[string]string{
		"@@USER_INITIAL_LASTNAME":          "{{.FirstNameInitial}}{{.LastName}}",
		"@@USER_LASTNAME_INITIAL":          "{{.LastName}}{{.FirstNameInitial}}",
		"@@USER_FIRSTNAME_INITIAL":         "{{.FirstName}}{{.LastNameInitial}}",
		"@@USER_FIRSTNAME.@@USER_LASTNAME": "{{.FirstName}}.{{.LastName}}",
		"@@USER_FIRSTNAME":                 "{{.FirstName}}",
		"@@" + cfg.EnvSSHUsername:          "{{.FullName}}",
	}
)

// UserEnv contains settings from the ENV
type UserEnv struct {
	User     string
	Simulate bool
}

// userName maps templated entries for usernames
type userName struct {
	FirstName, FirstNameInitial, FullName, LastName, LastNameInitial, Raw string
	IsParsed                                                              bool
}

// Connection stores the SSH properties
type Connection struct {
	HostName            string
	Port                uint16
	User                string
	LocalForward        []LocalForward
	IdentityFile        string
	IdentitiesOnly      bool
	ProxyJump           string
	ServerAliveInterval uint16
	ServerAliveCountMax uint16
	Cache               CachedConnection
	ControlPath         string
	//Compression bool
	//ControlMaster bool
	//ControlPersist uint16
}

// CachedConnection contains a full config
type CachedConnection struct {
	Config string
}

// LocalForward stores the port-forwarding details
type LocalForward struct {
	LocalPort  uint16
	RemotePort uint16
	BindHost   string
}

// acquirePort for LocalForward
// min : lowest port to choose
// max : highest port to choose
func acquirePort(min uint16, max uint16) (uint16, error) {
	for i := uint16(min); i <= max; i++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", i))
		if l != nil {
			l.Close()
		}
		if err != nil {
			continue
		}
		return i, nil
	}
	return 0, errNoFreePort
}

// doMarshal of userName to JSON format
func (un *userName) doMarshal() (string, error) {
	data, err := json.Marshal(un)

	if err != nil {
		log.Error("Failed to marshal:", un, ", error:", err)
		return "", err
	}
	return string(data), nil
}

// doUnmarshal of userName from JSON
// jsonString : userName in JSON format
func (un *userName) doUnmarshal(jsonString string) (userName, error) {
	var newUser userName

	err := json.Unmarshal([]byte(jsonString), &newUser)

	if err != nil {
		log.Error("Failed to unmarshal:", jsonString, ", error: ", err)
		return userName{}, err
	}
	return newUser, nil
}

// doUnmarshalToKeys of userName from JSON
// jsonString : userName in JSON format
func (un *userName) doUnmarshalToKeys(jsonString string) (map[string]interface{}, error) {
	var keyedItem map[string]interface{}

	err := json.Unmarshal([]byte(jsonString), &keyedItem)

	if err != nil {
		log.Error("Failed to unmarshal:", jsonString, ", error: ", err)
		return nil, err
	}
	return keyedItem, nil
}

// generateUserName converts a string to a userName
// username : The full name of the user, automatically split on period
func (un *userName) generateUserName(username string) (bool, error) {
	name := strings.Split(username, ".")
	un.IsParsed = false

	if len(un.FirstName) > 0 {
		return false, errors.New("rejecting request, userName already initialised")
	} else if strings.HasPrefix(username, "@") {
		un.FirstName = username
		un.FirstNameInitial = username
		un.LastName = username
		un.LastNameInitial = username
		un.FullName = username
	} else if len(name) > 1 {
		un.FirstName = name[0]
		un.FirstNameInitial = name[0][0:1]
		un.LastName = name[1]
		un.LastNameInitial = name[1][0:1]
		un.FullName = username

		if len(name) > 2 {
			log.Warningf("Username '%s' contains more than one period, only the first one is recognised", username)
		}
	} else {
		un.FirstName = username
		un.FullName = username
	}
	un.Raw = username
	return true, nil
}

// rewriteUsername config templates
func (un *userName) rewriteUsername(newuser string) (bool, error) {
	var b bytes.Buffer
	var tempUser = userName{}
	tempUser.generateUserName(newuser)
	log.Debugf("original user '%v'", un)

	if len(un.FirstName) == 0 {
		log.Warning("User has not been initialised")
		return false, errors.New("user has not been initialised")
	}

	jsonUser, err := un.doMarshal()
	if err != nil {
		log.Errorf("Unable to convert '%v' to JSON; err %v", un, err)
		return false, err
	}
	log.Debugf("jsonUser '%v", jsonUser)

	for marker, tpl := range Placeholders {
		if marker == "@@" {
			// Broken marker from misconfigured env
			continue
		}
		log.Debugf("rewriting marker '%v' with '%v'", marker, tpl)
		jsonUser = strings.Replace(jsonUser, marker, tpl, 1)
	}
	log.Debugf("jsonUser rewritten '%v", jsonUser)

	tpl, err := template.New("userName").Parse(jsonUser)
	if err != nil {
		log.Panicf("Unable to rewrite username: %v", un)
	}
	tpl.Execute(&b, tempUser)

	templatedUser, err := un.doUnmarshal(b.String())
	if err != nil {
		log.Errorf("Unable to process template '%v' to JSON; err %v", b, err)
		return false, err
	}
	log.Debugf("templatedUser '%v'", templatedUser)

	templatedUser.IsParsed = true
	*un = templatedUser
	log.Debugf("updated user '%v'", un)
	return true, nil
}

// setHostname specifies the HostName value for SSH
// sshArgs : Connection properties for SSH
// args : options provided for inspection
func setHostname(sshArgs *Connection, args map[string]interface{}) {
	option := "localhost"
	if val, ok := args["HostName"]; ok {
		option = val.(string)
	}
	sshArgs.HostName = option
}

// setUser specifies the User value for SSH
// sshArgs : Connection properties for SSH
// args : options provided for inspection
func setUser(sshArgs *Connection, args map[string]interface{}, templateUser string) {
	option := cfg.EnvSSHDefaultUsername
	log.Debugf("original user: %v", templateUser)
	if val, ok := args["User"]; ok {
		option = val.(string)
	}
	log.Debugf("loaded user: %v", option)

	tempUser := userName{}
	if _, err := tempUser.generateUserName(option); err != nil {
		log.Error("Unable to generate tempUser")
	}
	log.Debugf("tempUser: %v", tempUser)

	if _, err := tempUser.rewriteUsername(templateUser); err != nil {
		log.Error("Unable to rewrite tempUser")
	}
	log.Debugf("tempUser updated to: %v", tempUser)

	sshArgs.User = tempUser.FirstName
}

// args : options provided for inspection
func setPort(sshArgs *Connection, args map[string]interface{}) {
	option := uint16(22)
	if val, ok := args["Port"]; ok {
		portInt, err := strconv.ParseInt(val.(string), 10, 0)
		if err != nil {
			panic(err)
		}
		option = uint16(portInt)
	}
	sshArgs.Port = option
}

// setIdentity specifies the IdentityFile value for SSH
// sshArgs : Connection properties for SSH
// args : options provided for inspection
func setIdentity(sshArgs *Connection, args map[string]interface{}) {
	option := cfg.EnvSSHIdentityFile
	if val, ok := args["IdentityFile"]; ok {
		option = val.(string)
	}
	sshArgs.IdentityFile = fmt.Sprintf("IdentityFile=%s", option)
}

// setProxy specifies the ProxyJump value for SSH
// sshArgs : Connection properties for SSH
// args : options provided for inspection
func setProxy(sshArgs *Connection, args map[string]interface{}) {
	option := "none"
	if val, ok := args["ProxyJump"]; ok {
		option = val.(string)
	}
	sshArgs.ProxyJump = option
}

// setControlPath for the connection
func setControlPath(sshArgs *Connection, args map[string]interface{}) {
	log.Debug("setControlPath: ", sshArgs)
	option := "cp"

	for _, v := range []string{"User", "HostName", "Port"} {
		if val, ok := args[v]; ok {
			option += fmt.Sprintf("_%s", val.(string))
		} else {
			switch v {
			case "User":
				option += fmt.Sprintf("_%s", sshArgs.User)
			case "HostName":
				option += fmt.Sprintf("_%s", sshArgs.HostName)
			case "Port":
				option += fmt.Sprintf("_%d", sshArgs.Port)
			}
		}
	}
	if option == "cp" || option == "cp___" {
		option = "%r@%h:%p"
	}
	sshArgs.ControlPath = fmt.Sprintf("%s/%s", cfg.StoragePath, option)
}

// setPortForwarding for the connection
// sshArgs : Connection properties for SSH
func setPortForwarding(sshArgs *Connection) {
	var lf LocalForward
	var data map[string]interface{}

	_, err := os.Stat(sshArgs.ControlPath)
	if err == nil {
		log.Debug("ControlPath exists")
		read, err := ioutil.ReadFile(sshArgs.ControlPath + ".json")
		if err == nil {
			log.Debug("Reading cached port-forwards")
			if err := json.Unmarshal(read, &data); err == nil {
				for _, k := range []string{"NGINX", "PMM"} {
					log.Debug("Setting port for: ", k)
					if val, ok := data[k]; ok {
						p, _ := strconv.ParseUint(val.(string), 10, 0)
						_, err := acquirePort(uint16(p), uint16(p))
						if err != nil {
							log.Debugf("Found port '%v' in use, reusing", val)
							rp := 0
							switch k {
							case "NGINX":
								rp = 443
							case "PMM":
								rp = 8443
							}
							sshArgs.LocalForward = append(sshArgs.LocalForward, LocalForward{uint16(p), uint16(rp), "127.0.0.1"})
						}
					}
				}
				if len(sshArgs.LocalForward) == 2 {
					return
				}
				sshArgs.LocalForward = []LocalForward{}
			}
		}

	}
	p := localForwardPortMin
	data = map[string]interface{}{
		"NGINX": "",
		"PMM":   "",
	}

	for _, rp := range []uint16{443, 8443} {
		lp, err := acquirePort(p, localForwardPortMax)
		if err != nil {
			panic(err)
		}
		p = lp + 1
		lf = LocalForward{lp, rp, "127.0.0.1"}
		sshArgs.LocalForward = append(sshArgs.LocalForward, lf)
		key := ""
		switch rp {
		case 443:
			key = "NGINX"
		case 8443:
			key = "PMM"
		}
		data[key] = fmt.Sprintf("%d", lp)
	}

	buff, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Failed to generate JSON to cache '%v': %v", sshArgs.ControlPath, err)
	}

	if err := ioutil.WriteFile(sshArgs.ControlPath+".json", []byte(string(buff)), 0640); err != nil {
		log.Errorf("Failed to save cache for '%v': %v", sshArgs.ControlPath, err)
	}
}

// BuildConnection creates the SSH command for execution
// args : options provided for inspection
func (c *Connection) BuildConnection(args map[string]interface{}, key string, templateUser string) []string {
	var sshArgsList []string

	setUser(c, args, templateUser)
	setPort(c, args)
	setIdentity(c, args)
	setProxy(c, args)
	setHostname(c, args)
	setControlPath(c, args)
	setPortForwarding(c)

	d := reflect.ValueOf(c).Elem()
	t := d.Type()
	ind := "  "

	c.Cache.Config = fmt.Sprintln("Host", key)

	for i := 0; i < d.NumField(); i++ {
		f := d.Field(i)
		switch t.Field(i).Name {
		case "HostName":
			c.Cache.Config += fmt.Sprintln(ind, t.Field(i).Name, f.Interface())
			continue
		case "IdentitiesOnly":
		case "ServerAliveCountMax":
		case "ServerAliveInterval":
		case "Cache":
			continue
		case "IdentityFile":
			c.Cache.Config += fmt.Sprintln(ind, strings.Replace(f.Interface().(string), "=", " ", 1))
			c.Cache.Config += fmt.Sprintln(ind, "IdentitiesOnly yes")
			sshArgsList = append(sshArgsList, []string{
				"-o", "IdentitiesOnly=yes",
				"-o", f.Interface().(string),
			}...)
		case "LocalForward":
			for _, lf := range c.LocalForward {
				sshArgsList = append(sshArgsList, []string{
					"-L", fmt.Sprintf("%d:%s:%d", lf.LocalPort, lf.BindHost, lf.RemotePort),
				}...)
			}
		case "ProxyJump":
			c.Cache.Config += fmt.Sprintln(ind, t.Field(i).Name, f.Interface())
			if fmt.Sprintf("%s", f.Interface()) == "none" {
				continue
			}
			sshArgsList = append(sshArgsList, []string{
				"-o", fmt.Sprintf("%s=%s", t.Field(i).Name, f.Interface()),
			}...)
		case "Port":
			c.Cache.Config += fmt.Sprintln(ind, t.Field(i).Name, c.Port)
			sshArgsList = append(sshArgsList, []string{
				"-p", fmt.Sprintf("%d", c.Port),
			}...)
		default:
			c.Cache.Config += fmt.Sprintln(ind, t.Field(i).Name, f.Interface())
			sshArgsList = append(sshArgsList, []string{
				"-o", fmt.Sprintf("%s=%s", t.Field(i).Name, f.Interface()),
			}...)
		}
	}
	return append(sshArgsList, c.HostName)
}

// Connect executes the SSH command
// args : options provided for inspection
// e : user environment settings
func Connect(args []string, e UserEnv) {
	if e.Simulate {
		log.Println("cmd: ssh", strings.Join(args, " "))
	} else {
		cmd := exec.Command("ssh", args...)
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
