path "sys/*" {
  policy = "deny"
}

path "secret/ssh_ms/*" {
  policy = "read"
  capabilities = ["create", "read", "update", "patch", "delete", "list"]
}

path "secret/ssh_ms_kv1/*" {
  policy = "read"
  capabilities = ["create", "read", "update", "patch", "delete", "list"]
}

path "secret/ssh_ms_kv2/*" {
  policy = "read"
  capabilities = ["create", "read", "update", "patch", "delete", "list"]
}
