path "sys/*" {
  policy = "deny"
}

path "secret/ssh_ms/*" {
  policy = "read"
  capabilities = ["list", "sudo"]
}
