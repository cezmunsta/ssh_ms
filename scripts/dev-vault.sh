#!/bin/bash

set -eux

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

    ssh_ms write test --comment Testing HostName=localhost User=@@USER_FIRSTNAME
}

if ! vault_exists; then
    create_vault
fi

start_vault
