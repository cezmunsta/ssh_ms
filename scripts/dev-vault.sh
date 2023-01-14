#!/bin/bash

set -eux

declare -ir RUN="${1:-0}"

cd "$(dirname "${0}")/../" || exit 1

function vault_exists {
    podman container exists dev-vault
    return ${?}
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
    until podman container ls --filter name=dev-vault,status=running -q | grep -Eq \\w; do
      sleep 1;
    done

    set +x
    export VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN=myroottoken
    echo -n "${VAULT_TOKEN}" | vault login -no-store -non-interactive -
    set -x

    vault secrets disable secret/
    vault secrets enable --path=secret/ssh_ms kv
    vault secrets enable --path=moresecret/ssh_ms kv
    vault secrets enable --path=secret/ssh_ms_admin kv

    ./bin/ssh_ms write test --comment Testing HostName=localhost User=@@USER_FIRSTNAME
}

function create_policy {
    local policy_name="${1}"
    local policy_src="${2}"

    cat "${policy_src}" | vault policy write "${policy_name}" -
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

test "${RUN}" -eq 0 || {
    if ! vault_exists; then
        create_vault
    fi

    start_vault
    create_policy ssh_ms_admin templates/policy-full.sh
    create_policy ssh_ms templates/policy-min.sh
    add_user
}
