path "sys/*" {
  policy = "deny"
}

path "secret/ssh_ms/*" {
  policy = "read"
  capabilities = ["list", "sudo"]
}

path "moresecret/ssh_ms/*" {
  policy = "read"
  capabilities = ["list", "sudo"]
}

path "secret/ssh_ms_admin/*" {
  policy = "read"
  capabilities = ["list", "sudo"]
}
