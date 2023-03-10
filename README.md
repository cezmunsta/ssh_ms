![](https://github.com/cezmunsta/ssh_ms/actions/workflows/pr-go.yaml/badge.svg)
[![CodeQL](https://github.com/cezmunsta/ssh_ms/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/cezmunsta/ssh_ms/actions/workflows/codeql-analysis.yml)
[![Scorecard supply-chain security](https://github.com/cezmunsta/ssh_ms/actions/workflows/scorecard.yml/badge.svg)](https://github.com/cezmunsta/ssh_ms/actions/workflows/scorecard.yml)
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
$ go install github.com/cezmunsta/ssh_ms
```

### Recommended versions:

For ease of use, ensure that `${GOPATH}/bin` is in your `PATH` to use the tools with ease.

- `go` : `1.18.9`
- `vault`: `1.12.2`

#### Go

Once you have a version installed then you can follow the documentation to
[install extra versions](https://golang.org/doc/install#extra_versions).

#### Vault

In order to use the more secure authentication approach, you will also need to install `Vault`.
Binaries are available from the [official Vault download page](https://www.vaultproject.io/downloads/).

## Usage

```sh
Usage:
  ssh_ms [flags]
  ssh_ms [command]

Available Commands:
  cache       Cache management
  completion  Generate completion script
  connect     Connect to a host
  delete      Delete a connection
  help        Help about any command
  inspect     Inspect the value of an internal item
  list        List available connections
  print       Print out the SSH command for a connection
  search      Search for a connection
  show        Display a connection
  update      Update an existing connection to storage
  version     Show the version
  write       Add a new connection to storage

Flags:
  -d, --debug                Provide addition output
  -n, --dry-run              Prevent certain commands without full execution
  -h, --help                 help for ssh_ms
  -s, --storage string       Storage path for caching (default "/home/user/.ssh/cache")
      --stored-token         Use a stored token from 'vault login' (overrides --vault-token, auto-enabled when no token is specified)
  -u, --user string          Your SSH username for templated configs (default "user")
      --vault-addr string    Specify the Vault address (default "http://127.0.0.1:8200")
      --vault-token string   Specify the Vault token (default "myroottoken")
  -v, --verbose              Provide addition output

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

### Using templated usernames
When either using in a shared environment, or when wishing to reuse a connection with a choice of User values, templated entries are supported.
You can view the supported templates using the `inspect` command, e.g.
```shell
$ ssh_ms inspect placeholders
@@USER_FIRSTNAME.@@USER_LASTNAME
@@USER_FIRSTNAME
@@SSH_MS_USERNAME
@@USER_INITIAL_LASTNAME
@@USER_LASTNAME_INITIAL
@@USER_FIRSTNAME_INITIAL
```

When using `--verbose` mode you will also see the templates associated with these:
```shell
$ ssh_ms inspect placeholders -v
@@USER_LASTNAME_INITIAL = {{.LastName}}{{.FirstNameInitial}}
@@USER_FIRSTNAME_INITIAL = {{.FirstName}}{{.LastNameInitial}}
@@USER_FIRSTNAME.@@USER_LASTNAME = {{.FirstName}}.{{.LastName}}
@@USER_FIRSTNAME = {{.FirstName}}
@@SSH_MS_USERNAME = {{.FullName}}
@@USER_INITIAL_LASTNAME = {{.FirstNameInitial}}{{.LastName}}
```

Using templated usernames will require either an environment variable to be set (`SSH_MS_USERNAME` by default), or by using the `--user` argument.
Writing these to storage is done the same way as any other connection, except that you specify the template instead of the real user:
```shell
$ ssh_ms write test User=@@USER_LASTNAME_INITIAL
```


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
localhost testing gateway-us-1

$ ssh_ms search local
localhost

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
$ ssh_ms purge
```

### Setting comments and message of the day
To help identify connections, or to give some extra information when a user connects, you can set
a comment and/or a message of the day that will be displayed. When these are not set there will still
be a sort of motd that displays the current connection and any port forwarding.
```sh
$ ssh_ms update testing --comment "This is a comment" --motd "This is the motd"
$ ssh_ms show testing
# This is a comment
Host testing
   HostName localhost
   Port 22
   User bob
   IdentityFile ~/.ssh/custom_rsa
   IdentitiesOnly yes
   ProxyJump none
   ControlPath /home/bob/.ssh/cache/cp_bob_localhost_22

$ ssh_ms connect testing date

***************************************************************
# This is a comment
Server connection: testing

This is the motd

FWD: https://127.0.0.1:18000 - NGINX (443)
FWD: https://127.0.0.1:18001 - PMM (8443)
***************************************************************

Wed 19 May 17:20:44 BST 2021
Shared connection to localhost closed.
```

### Using namespaces
It may be desirable to maintain multiple namespaces in Vault, so that access to specific connections can be
controlled, such as a single binary that can be used by users with different policies applied to their account.
Namespaces are configured at build time via the `SSH_MS_SECRET_PATH` variable. When performing write operations
without specifying a namespace, a default one will be chosen (the first one in the list of namespaces).

```shell
# Check which namespaces are available
$ ssh_ms version --verbose
Version: 8a8d6a915e13999bef6fb6b4bd279459d743ce9c
Arch: linux amd64
Go Version: go1.18.6
Vault Version: 1.12.0
Base path: /home/user/.ssh/cache
Namespaces:
- secret/ssh_ms
- secret/my-special-namespace
Default Vault address: http://127.0.0.1:8200
Default SSH username: user
SSH template username: SSH_MS_USERNAME
SSH identity file: ~/.ssh/id_ed25519

# Add a connection to a non-default namespace,
$ ssh_ms write secret-connection --namespace secret/my-special-namespace

# Search and list will scan all available namespaces
$ ssh_ms list
test secret-connection

$ ssh_ms search conn
secret-connection

# List a specific namespace
$ ssh_ms list --namespace secret/my-special-namespace
secret-connection
```

## Build

Should you wish to build the binary to have some defaults preset for you, then you can use the following env variables
along with `make build`:
- `BUILD_DIR` : Set the location for the binary
- `RELEASE_VER` : Sets `cmd.Version`
- `SSH_MS_BASEPATH`: Sets `config.EnvBasePath`
- `SSH_MS_DEFAULT_VAULT_ADDR`: Sets `config.EnvVaultAddr`, bypassing environment lookup of `VAULT_ADDR`
- `SSH_MS_DEFAULT_USERNAME`: Sets `config.EnvSSHDefaultUsername`, bypassing environment lookup of `USER`
- `SSH_MS_ID_FILE`:  Sets `config.EnvSSHIdentityFile`
- `SSH_MS_RENEW_THRESHOLD`: Sets `vault.RenewThreshold`
- `SSH_MS_SECRET_PATH`: Sets the searchable paths in Vault
- `SSH_MS_SYNC_HOST`: Sets the destination host for a binary push via `rsync`
- `SSH_MS_SYNC_PATH`: Sets the destination path for a binary push via `rsync`
- `SSH_MS_USERNAME`: Sets `config.EnvSSHUsername` template variable, used in templated usernames

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

## Development environment

To create your own local Vault for use during development, perform the following steps:
```sh
$ podman run -d --cap-add=IPC_LOCK --name=dev-vault --network=host \
    -e VAULT_DEV_ROOT_TOKEN_ID=myroottoken -e VAULT_DEV_LISTEN_ADDRESS=127.0.0.1:8200 vault

$ export VAULT_ADDR=http://127.0.0.1:8200
$ export VAULT_TOKEN=myroottoken

$ vault login -no-store
$ vault secrets disable secret/
$ vault secrets enable --path=secret/ssh_ms kv

$ ssh_ms write test --comment Testing HostName=localhost User=@@USER_FIRSTNAME
```

If you wish to run the server in production mode, which would require the `Cmd`
being adjusted to remove the `-dev` flag, then you will most likely want to use some
persistent volumes:

```sh
$ podman volume create vault-file
$ podman volume create vault-logs
$ podman run -d --cap-add=IPC_LOCK --name=dev-vault --network=host \
     -e VAULT_DEV_ROOT_TOKEN_ID=myroottoken -e VAULT_DEV_LISTEN_ADDRESS=127.0.0.1:8200 \
     -v vault-file:/vault/file -v vault-logs:/vault/logs vault server <your options>
```
