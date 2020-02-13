# ssh_ms

Integrate with [HashiCorp Vault](https://github.com/hashicorp/vault) to store SSH configs and ease your remote life.
You will no longer need to make changes to your local `.ssh/config`, except if you need gateway hosts, and you will
have access to the same configs from anywhere. All of this while you are safe with the knowledge that they
require authentication in order to access them.

## Installation

Use of this tool requires a working Vault installation, which is out of the scope of this documentation. Please see
the official [HashiCorp documentation](https://www.vaultproject.io/docs/) for more information.

In order to install the tool using `go get` you will first require a working installation
of `Go`. Please follow the [official documentation](https://golang.org/doc/install) for
installing `Go`.

```sh
$ go get -u github.com/cezmunsta/ssh_ms
```

### Recommended versions:

For ease of use, ensure that `${GOPATH}/bin` is in your `PATH` to use the tools with ease.

- `go` : `1.13.7`
- `vault`: `1.3.2`

#### Go

Once you have a version installed then you can follow the documentation to
[install extra versions](https://golang.org/doc/install#extra_versions).

#### Vault

In order to use the more secure authentication approach, you will also need to install `Vault`.
Binaries are available from the [official Vault download page](https://www.vaultproject.io/downloads/).

## Usage

```sh
ssh_ms integrates with HashiCorp Vault to store SSH configs and ease your remote life

Usage:
  ssh_ms [flags]
  ssh_ms [command]

Available Commands:
  connect     Connect via SSH
  help        Help about any command
  list        List available connections
  purge       Purge the cache
  show        Display a connection
  write       Write, or update a config

Flags:
  -n, --dry-run              Display commands rather than executing them
  -h, --help                 help for ssh_ms
  -s, --storage string       Storage path for caching (default "~/.ssh/cache")
      --stored-token         Use a stored token from 'vault login' (overrides --vault-token)
  -u, --user string          Your SSH username for templated configs
      --vault-addr string    Specify the Vault address
      --vault-token string   Specify the Vault token
  -v, --verbose              Provide addition output
  -V, --version              Show the version

Use "ssh_ms [command] --help" for more information about a command.
```

## Examples

### Authenticating
In order to use the tool you will need the ability to authenticated against HashiCorp Vault.
If you have installed the `vault` binary then you will be able to use the more secure authentication
method using the `--stored-token` argument.

First, you should authenticate against `Vault`, which when using the default method is as follows:
```sh
VAULT_ADDR=https://127.0.0.1:8200 vault login
```
You can change the method for authentication using `-method` (see `vault login --help` for more info).

After you have authenticated you will then be able to use `ssh_ms` without specifying your token, e.g.
```sh
VAULT_ADDR=https://127.0.0.1:8200 ssh_ms list --stored-token
# or
ssh_ms list --vault-addr https://127.0.0.1:8200 --stored-token
```

**N.B.** If both the `--vault-token` argument and `VAULT_TOKEN` are unset then `--stored-token` is automatically
applied and can be omitted from commands

It is also possible to use either the `VAULT_TOKEN` environment variable, or `--vault-token` to specify
your token should you need to. Whenever possible, the pre-authenticated, more secure method should be
used along with `--stored-token`.

### Add a gateway

To make life easier, you may wish to export the address for Vault in your shell profile:
```sh
export VAULT_ADDR=https://127.0.0.1:8200
```

The following examples will presume that this is the case, as well as defaulting to using the stored token:

```sh
$ ssh_ms write gateway-us-1 HostName=192.168.0.1 IdentityFile='~/.ssh/custom_rsa' User='@@SSH_USER'
```

Using the `show` command you can then pull the config so that you can write it locally:

#### Default template (placeholder is removed from view)
```sh
$ ssh_ms show gateway-us-1
Host gateway-us-1
   HostName 192.168.0.1
   Port 22
   User
   IdentityFile ~/.ssh/custom_rsa
   IdentitiesOnly yes
   ProxyJump none
```

#### Specify username via arguments
```sh
$ ssh_ms show gateway-us-1 --user bob
Host gateway-us-1
   HostName 192.168.0.1
   Port 22
   User bob
   IdentityFile ~/.ssh/custom_rsa
   IdentitiesOnly yes
```

#### Specify username via environment variable
```sh
$ SSH_USERNAME=bob ssh_ms show gateway-us-1
Host gateway-us-1
   HostName 192.168.0.1
   Port 22
   User bob
   IdentityFile ~/.ssh/custom_rsa
   IdentitiesOnly yes
```

### Finding available connections
```sh
$ ssh_ms list
available connections:
localhost testing gateway-us-1

$ ssh_ms show localhost
Host localhost
   HostName 127.0.0.1
   Port 22
   User bob
   IdentityFile ~/.ssh/custom_rsa
   IdentitiesOnly yes
   ProxyJump none
```

### Connecting
```sh
$ ssh_ms connect localhost date
Wed 28 Mar 16:39:46 BST 2018

$ ssh_ms connect testing
Last login: Sat Feb  8 06:21:08 2020 from 192.168.0.1
bob@testing: ~ $
```

### Purge your cache
Each connection that is retrieved from `Vault` is cached locally for 1 week. Should you need to
force this to be cleared then you can use the `purge` command:
```sh
ssh_ms purge
```

## Build

Should you wish to build the binary to have some defaults preset for you, then you can use the following env variables
along with `make build`:
- `BUILD_DIR` : Set the location for the binary
- `RELEASE_VER` : Sets `cmd.Version`
- `DEFAULT_VAULT_ADDR` : Sets `cmd.EnvVaultAddr`
- `SSH_DEFAULT_USERNAME` : Sets `ssh.EnvSSHDefaultUsername`
- `SSH_MS_USERNAME` : Sets `cmd.EnvSSHUsername`
- `SSH_ID_FILE` : Sets `cmd.EnvSSHIdentityFile`

To set the build version (e.g. if you are making a custom build, etc) you can use `make build` to set
the build version (by default it will set it to `git rev-parse HEAD`), e.g.
```sh
make build -e BUILD_VER=1.0.0-alpha
```

Alternatively, you can build directly and use `-ldflags`. For example, here we will set the build
version to 0.2 and build the binary as builds/ssh_ms:
```sh
$ go build -ldflags "-X github.com/cezmunsta/ssh_ms/cmd.Version=0.2" -o builds/ssh_ms
```


