#!/bin/bash

set -eux

declare -ir RUN="${1:-1}"

function vault_exists {
    # shellcheck disable=SC2155
    local id="$(podman container ls --filter name=dev-vault --all --quiet)"
    local -i status=1

    test -z "${id}" || status=0
    return ${status}
}

function create_vault {
    podman container create --cap-add=IPC_LOCK --name=dev-vault --network=host \
        -e VAULT_DEV_ROOT_TOKEN_ID=myroottoken \
        -e VAULT_DEV_LISTEN_ADDRESS=127.0.0.1:8200 vault
}

function start_vault {
    podman container start dev-vault
    prepare_vault
    if [ "${DEBUG:-0}" -eq 1 ]; then
        watch_vault
    fi
}

function watch_vault {
    podman container logs --follow
}

function prepare_vault {
    export VAULT_ADDR=http://127.0.0.1:8200 \
           VAULT_TOKEN=myroottoken

    vault login -no-store
    vault secrets disable secret/
    vault secrets enable --path=secret/ssh_ms kv
    vault secrets enable --path=moresecret/ssh_ms kv
    vault secrets enable --path=secret/ssh_ms_admin kv

    ./bin/ssh_ms write test --comment Testing HostName=localhost User=@@USER_FIRSTNAME
}

function create_policy {
    vault policy write ssh_ms <(cat <<EOS
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
EOS
    )
}

function add_user {
    vault auth enable userpass
    vault write auth/userpass/users/testing \
        password=my-secret-pw \
        policies=default,ssh_ms renewable=true ttl=2h
}

function login_test_user {
    vault login --method userpass username=testing
}

function renew_test_token {
    vault token lookup --format=yaml | grep -Fq 'display_name: userpass-testing' || \
        echo 'Please login as the test user'
        return
    vault token renew --increment=30m
}

test "${RUN}" -eq 1 && {
    if ! vault_exists; then
        create_vault
    fi

    start_vault
    create_policy
    add_user
}
