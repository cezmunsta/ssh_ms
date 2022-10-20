# CHANGELOG

## 2022-10-20

- Update Vault to v1.12.0 (#97)
- Add support for multiple secret namespaces (#96)

## 2021-11-28: v1.7.1

- Updated goutils to v1.1.1 (#93)
- Updated Vault to 1.11.1 (#92)
- Updated cobra@v1.4.0 (#91)
- PR workflow improvements (#89)
- Updated to Go 1.18 (#87)
- Updated dependencies (#82)


## 2021-11-28: v1.7.0

- Notify user when their token will soon expire (#81)
  To help avoid unexpected expiration of tokens, the user is provided with a warning when they use a renewable token and it is due to expire in less than 7 days (default).
  The threshold for notifying about renewing tokens, `SSH_MS_RENEW_THRESHOLD` can be defined for `make build` and `make binaries`.
- Use only vault/api in application code (#80)
  To reduce size as well as simply issues arising from indirect dependencies, replacing the use of HashiCorp `vault/command` with the `api` in the helper code.
- Add support for SendEnv (#76)
  In cases where the remote server supports environment variables being passed across, adding support for storing `SendEnv` in the connection's configuration
- Updated dependencies (#75)
  - `vault` to v1.8.5
  - `vault/sdk` to v0.2.2-0.20211101151547-6654f4b913f9
  - `mimetype` to v1.4.0
  - `logrus` to v1.8.1
- Fix incorrect conversion between integer types (#74)
  Updated NGINX and PMM ports to become uint16 and switched to `strconv.ParseUint`
- Updated README
  Added the CodeQL badge and updated the recommended version of Vault
- Adding CodeQL workflow
- Upgraded Vault to 1.8.4 (#73)

## 2021-10-14: v1.6.0

- Added cache management (#72)
  A new command, `cache`, has been created with subcommands for supported operations on the cache, which currently is limited to populating and purging.
  The `purge` command has been replaced by `cache purge`
- Fix bad switch in cmd.inspectItem (#70)

## 2021-09-21: v1.5.0

- Add option to view usable placeholders for User (#68)
  Adding an option for the user to list the available ones makes the use
  of templated users easier
  ```shell
  $ ssh_ms inspect placeholders
  ```
- Hash ControlPath socket names by default (#66)
  Currently, the dynamic ControlPath is done in such a way as to make it easy to determine its purpose. However, should long HostName fields exist then this could potentially exceed the maximum path length for a UNIX socket (UNIX_PATH_MAX). By switching to using a hash, similar to %C in ssh, we can restrict the length of the path
- Moved go get golint to separate task

## 2021-09-06: v1.4.0

- Upgrade Vault and Logrus (#64)
  Vault has been upgraded to v1.8.2 and Logrus to v1.7.0
- Add support for ForwardAgent (#62)
  Whilst `ForwardAgent` is normally disabled for security reasons, there
  are certain circumstances where it is required. An example of
  required usage is where a third-party requires 2FA and a \
  certificate and key are injected into the user’s ssh-agent upon
  successful authentication.
- Adding PR workflow (#63)
- Updated Vault to v1.8.1 (#61)
- Added push workflow for Go source code (#60)
- Remove warning during write (#59)
  When writing a new connection, an unnecessary warning appeared:
  ```shell
  level=warning msg="Unable to find connection for: xxx"
  ```
  This is no longer shown.

## 2021-07-29: v1.3.0

- Extra information for versionCmd (#56)
  The Go and Vault versions are now shown when using `version --verbose`
- Update Vault dependencies (#55)
  Upgraded Vault to v1.8.0
- Add option to check for the latest release (#52)
  The user is now able to check for the latest release with `version --check`
- Enable cmd.TestCache (#50)
  Caching is now tested during `cmd` tests
- Ignore misses for lock requests (#49)
  Due to the locking mechanism sharing code with standard requests, warning messages
  were always emitted during a request when the lock is absent (ideal state). These
  are now hidden based upon the lock prefix


## 2021-06-21: v1.2.2

- Handle tilde in config.EnvBasePath (#47)
  The tilde from the build option is not being parsed before use


## 2021-06-15: v1.2.1

- Ensure EnvBasePath exists (#45)
  Fixes the issue where the storage path is absent and is not automatically created

## 2021-06-15: v1.2.0

- Added missing entries from the changelog (#43)
- Fix override variables that aren't strings (#42)
  Some of the overrides were no longer working due to being defined in a way other than as an
  explicit string, which caused issues when building with overrides.
- Added support for message of the day (#37)
  A "message of the day" can now be added to the stored configuration, allowing messages to
  be displayed during the connection phase, including whatever relevant information is necessary.
  This also allows the message to be managed without accessing an instance, which is where the motd
  would normally be set; on-host motd messaging is not affected by this feature
- Updated Go-based tasks in Makefile (#36)
- Added extra tests to Makefile (#35)
- Added Vault tests (#34)
  Vault TestCluster has now been integrated into the test suits, allowing tests
  to run without access to a running Vault instance
- Update log level for messages (#33)
  Changed levels for some getConnections messages


## 2020-05-08: v1.1.0

- Updated README (#32)
- Added dynamic ControlPath definition (#31):
  In order to solve the problem of unnecessary `LocalForward` definitions
  when creating multiple connections to the same host, a scenario that
  occurs when a control path is used, specifying the `ControlPath` dynamically
  allows detection of an active connection. When the first connection is created
  the `ControlPath` is generated by SSH and we save the ports in the cache
  directory. For the next connection, if the `Controlpath` is still in existence
  then we can specify identical `LocalForward` entries without an issue.
- Added locking mechanism for write operations (#30):
  In multi-user environments it is possible that more than one user attempts to perform
  operations against the same key in Vault storage. The user's operation must now
  acquire a lock to be able to perform a write operation against the storage layer
- Add connection search (#29):
  The user can now `search` the existing list of connection using partial patterns,
  or even regular expressions; partial expressions must still compile as a regex
- Added argument checker for better UX (#28):
  Some basic argument checking is performed to help avoid common issues and
  aborting early on in the execution process.
- Enhance caching (#27):
  Caching operations and updates now take part when performing write operations
  instead of only when requesting a connection for use. The normal cache expiry
  operations take part during this process.
- Added support for representing the config in JSON format (#26):
  For use internally, the config can now be converted to JSON by calling the
  `Settings.ToJSON` function.
- Added dev-vault to Makefile (#25):
  A test Vault container can be created and unlocked using `make dev-vault`
- Partial updates (#24):
  The user can now apply an update to an existing connection by using `update`
  instead of `write`. An error will now occur when trying to use `write` with
  an existing entry, or trying to use `update` with a non-existent one.
- Major refactor of code (#22):
  Extensive code rewrite to solve some problems that arose when adding new
  features and fixing some bugs.

Please see [README.md](README.md) for more details.

## 2021-01-25: v1.0.1

- Makefile improvements (#17):
  Various improvements relating to build operations.
- Enable comments to be applied to a rendered config (#15):
  Added `--comment` to enable users to add contextual information, useful when
  generating content for `~/.ssh/config`, etc
- Format port forwarding links to allow "open link" (#16):
  HTTP links are generated, which the user's terminal should interpret and
  allow them to open in their browser.
- Force xz compression (#14):
  Use `-f` when compressing the binaries so as to be able to avoid
  extra calls to purge beforehand.
- Added shell completion support (#13):
  Initial support for generating shell completion.
- Improved builds via Makefile (#11):
  Support has been added to build both Linux and MacOS binaries and
  optionally rsync them to target destination for downloading.
- Refactor vault.WriteSecret (#9):
  `vault.WriteSecret` is now aligned with the other helpers. It now accepts a
  preformatted path instead of just the key.
- Add option to delete entries (#8):
  User may now remove entries without the need for direct use of the Vault client.

Please see [README.md](README.md) for more details.

## 2020-02-14: v1.0.0

First release version of ssh_ms, including the following features:
- Connect to a remote host using a shared configuration from Vault
- Writing SSH configuration to Vault
- Listing existing configurations
- Show a configuration to allow redirection to ~/.ssh/config, etc
- Local caching of configurations (1w ttl)
- Integration with vault login to use stored token

Please see [README.md](README.md) for more details.
