package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"

	"github.com/cezmunsta/ssh_ms/log"
)

var (
	errNoFreePort       = fmt.Errorf("no free port")
	localForwardPortMin = uint16(18000)
	localForwardPortMax = uint16(20000)

	// Placeholders are used for templated connections
	Placeholders = map[string]string{
		"@@USER_INITIAL_LASTNAME":  "{{.Firstinitial}}{{.Lastname}}",
		"@@USER_LASTNAME_INITIAL":  "{{.Lastname}}{{.Firstinitial}}",
		"@@USER_FIRSTNAME_INITIAL": "{{.Firstname}}{{.Lastinitial}}",
		"@@USER_FIRSTNAME":         "{{.Firstname}}",
		//"@@" + cmd.EnvSSHUsername:  "{{.Fullname}}",
	}

	// EnvSSHDefaultUsername is used to authenticate with SSH
	EnvSSHDefaultUsername = os.Getenv("USER")
)

// UserEnv contains settings from the ENV
type UserEnv struct {
	User     string
	Simulate bool
}

// userName maps templated entries for usernames
type userName struct {
	FirstName, FirstNameInitial, FullName, LastName, LastNameInitial string
	IsParsed                                                         bool
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
	//Compression bool
	//ControlMaster bool
	//ControlPath string
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

// generateUserName converts a string to a userName
func (un *userName) generateUserName(username string) (bool, error) {
	name := strings.Split(username, ".")

	if len(un.FirstName) > 0 {
		return false, errors.New("Rejecting request, userName already initialised")
	} else if len(name) > 1 {
		un.FirstName = name[0]
		un.FirstNameInitial = name[0][0:1]
		un.LastName = name[1]
		un.LastNameInitial = name[1][0:1]
		un.FullName = username
		un.IsParsed = true
	} else {
		un.FirstName = username
		un.FullName = username
		un.IsParsed = false
	}
	return un.IsParsed, nil
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
func setUser(sshArgs *Connection, args map[string]interface{}) {
	option := EnvSSHDefaultUsername
	if val, ok := args["User"]; ok {
		option = val.(string)
	}
	sshArgs.User = option
}

// setPort specifies the Port value for SSH
// sshArgs : Connection properties for SSH
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
	option := "~/.ssh/id_rsa"
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

// setPortForwarding for the connection
// sshArgs : Connection properties for SSH
func setPortForwarding(sshArgs *Connection) {
	var lf LocalForward
	p := localForwardPortMin

	for _, rp := range []uint16{443, 8443} {
		lp, err := acquirePort(p, localForwardPortMax)
		if err != nil {
			panic(err)
		}
		p = lp + 1
		lf = LocalForward{lp, rp, "127.0.0.1"}
		sshArgs.LocalForward = append(sshArgs.LocalForward, lf)
	}
}

// BuildConnection creates the SSH command for execution
// args : options provided for inspection
func (c *Connection) BuildConnection(args map[string]interface{}, key string) []string {
	var sshArgsList []string

	setUser(c, args)
	setPort(c, args)
	setIdentity(c, args)
	setProxy(c, args)
	setHostname(c, args)
	setPortForwarding(c)

	d := reflect.ValueOf(&*c).Elem()
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
