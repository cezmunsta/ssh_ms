# CHANGELOG

## 2020-02-13: v1.0.0

First release version of ssh_ms, including the following features:
- connect to a remote host using a shared configuration from Vault
- writing SSH configuration to Vault
- listing existing configurations
- show a configuration to allow redirection to ~/.ssh/config, etc
- local caching of configurations (1w ttl)
- integration with vault login to use stored token

Please see [README.md](https://github.com/cezmunsta/ssh_ms/blob/v1.0/README.md) for more details.
