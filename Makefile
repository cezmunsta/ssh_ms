# vim: ts=8:sw=8:ft=make:noai:noet
SHELL=/bin/bash

BUILD_DIR?="./bin"
RELEASE_VER?="`git rev-parse HEAD`"

DEFAULT_VAULT_ADDR?="https://127.0.0.1:8200"

SSH_DEFAULT_USERNAME?="${USER}"
SSH_MS_USERNAME?="SSH_MS_USERNAME"
SSH_ID_FILE?="id_rsa"
SSH_MS_SYNC_HOST?="localhost"
SSH_MS_SYNC_PATH?="/usr/share/nginx/html/downloads/ssh_ms/"

PACKAGE="github.com/cezmunsta/ssh_ms"
LDFLAGS=-ldflags "-w -X ${PACKAGE}/cmd.Version=${RELEASE_VER} -X ${PACKAGE}/cmd.EnvSSHUsername=${SSH_MS_USERNAME} -X ${PACKAGE}/cmd.EnvSSHIdentityFile=${SSH_ID_FILE} -X ${PACKAGE}/ssh.EnvSSHDefaultUsername=${SSH_DEFAULT_USERNAME} -X ${PACKAGE}/cmd.EnvVaultAddr=${DEFAULT_VAULT_ADDR}"

all: lint format test binaries

binaries: binary-linux binary-mac

flags:
	@echo -e "\"${LDFLAGS}\"" | sed 's/-ldflags /-ldflags "/; s/^"//'

sync:
	@rsync -rlpDvc --progress bin/{linux,darwin} "${SSH_MS_SYNC_HOST}":"${SSH_MS_SYNC_PATH}"

binary-prep:
	@mkdir -p ${BUILD_DIR}/${GOOS}/${GOARCH};

binary-mac: export GOOS=darwin
binary-mac: export GOARCH=amd64
binary-mac: binary-prep
	@go build -o "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms" ${LDFLAGS};
	@xz -fkez9 "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms";

binary-linux: export GOOS=linux
binary-linux: export GOARCH=amd64
binary-linux: binary-prep
	@go build -o "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms" ${LDFLAGS};
	@xz -fkez9 "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms";

build: binary-prep
	@go build -o "${BUILD_DIR}/ssh_ms" ${LDFLAGS}

test:
	@go test "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault"

lint:
	@golint ssh vault cmd

format:
	@gofmt -w ssh vault cmd

simplify:
	@gofmt -s -w ssh vault cmd

vet:
	@go vet "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault"

fix:
	@go tool fix -diff ssh vault cmd

clean:
	@find "${BUILD_DIR}" -type f -delete
