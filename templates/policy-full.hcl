path "sys/*" {
  capabilities = ["deny"]
}

path "secret/ssh_ms*" {
  capabilities = ["create", "read", "update", "patch", "delete", "list", "sudo"]
}
