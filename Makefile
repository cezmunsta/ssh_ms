# vim: ts=8:sw=8:ft=make:noai:noet
SHELL=/bin/bash

BUILD_DIR?="${GOPATH}/bin"
RELEASE_VER?="`git rev-parse HEAD`"

DEFAULT_VAULT_ADDR?="https://127.0.0.1:8200"

SSH_DEFAULT_USERNAME?="${USER}"
SSH_MS_USERNAME?="SSH_MS_USERNAME"
SSH_ID_FILE?="id_rsa"

PACKAGE="github.com/cezmunsta/ssh_ms"
LDFLAGS=-ldflags "-X ${PACKAGE}/cmd.Version=${RELEASE_VER} -X ${PACKAGE}/cmd.EnvSSHUsername=${SSH_MS_USERNAME} -X ${PACKAGE}/cmd.EnvSSHIdentityFile=${SSH_ID_FILE} -X ${PACKAGE}/ssh.EnvSSHDefaultUsername=${SSH_DEFAULT_USERNAME} -X ${PACKAGE}/cmd.EnvVaultAddr=${DEFAULT_VAULT_ADDR}"

all: lint format build

build:
	@go build -o "${BUILD_DIR}/ssh_ms" ${LDFLAGS}

lint:
	@golint .

format:
	@gofmt -w .

simplify:
	@gofmt -s -w .

vet:
	@go vet -x .

clean:
	@rm -f "${BUILD_DIR}/ssh_ms"
