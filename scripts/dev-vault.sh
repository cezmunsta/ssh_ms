#!/bin/bash

set -eux

declare -ir RUN="${1:-0}"

cd "$(dirname "${0}")/../" || exit 1

function vault_exists {
    podman container exists dev-vault
    return ${?}
}

function create_vault {
    podman container create --cap-add=IPC_LOCK,SETFCAP --name=dev-vault --network=host \
        -e VAULT_DEV_ROOT_TOKEN_ID=myroottoken \
        -e VAULT_DEV_LISTEN_ADDRESS=127.0.0.1:8200 hashicorp/vault
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
    local -r vault_host=127.0.0.1
    local -ri vault_port=8200
    local -r vault_scheme='http'
    local -A secrets=( [ssh_ms]="1" [ssh_ms_kv1]="1" [ssh_ms_kv2]="2" )

    set +x
    export VAULT_ADDR="${vault_scheme}://${vault_host}:${vault_port}" \
            VAULT_TOKEN=myroottoken

    until podman container ls --filter name=dev-vault,status=running -q | grep -Eq \\w; do
      sleep 1;
    done

    until nc -z "${vault_host}" "${vault_port}"; do
        sleep 1
    done

    echo -n "${VAULT_TOKEN}" | vault login -no-store -non-interactive -field token_duration -
    set -x

    vault secrets disable secret/

    for secret in "${!secrets[@]}"; do
        vault secrets enable --path=secret/"${secret}" --version="${secrets[${secret}]}" kv
        vault kv put --mount=secret/"${secret}" test HostName="${secret}-${secrets[${secret}]}.localhost"
    done
}

function create_policy {
    local policy_name="${1}"
    local policy_src="${2}"

    vault policy write "${policy_name}" - < "${policy_src}"
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
    create_policy ssh_ms_admin templates/policy-full.hcl
    create_policy ssh_ms templates/policy-min.hcl
    add_user
}
