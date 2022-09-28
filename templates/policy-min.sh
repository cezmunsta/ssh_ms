path "sys/*" {
  policy = "deny"
}

path "secret/ssh_ms/*" {
  policy = "read"
  capabilities = ["create", "read", "update", "patch", "delete", "list"]
}
