# vim: ts=8:sw=8:ft=make:noai:noet
SHELL=/bin/bash

BUILD_DIR?="./bin"
RELEASE_VER?="`git rev-parse HEAD`"

DEFAULT_VAULT_ADDR?="https://127.0.0.1:8200"

SSH_DEFAULT_USERNAME?="${USER}"
SSH_MS_USERNAME?="SSH_MS_USERNAME"
SSH_ID_FILE?="id_rsa"

PACKAGE="github.com/cezmunsta/ssh_ms"
LDFLAGS=-ldflags "-X ${PACKAGE}/cmd.Version=${RELEASE_VER} -X ${PACKAGE}/cmd.EnvSSHUsername=${SSH_MS_USERNAME} -X ${PACKAGE}/cmd.EnvSSHIdentityFile=${SSH_ID_FILE} -X ${PACKAGE}/ssh.EnvSSHDefaultUsername=${SSH_DEFAULT_USERNAME} -X ${PACKAGE}/cmd.EnvVaultAddr=${DEFAULT_VAULT_ADDR}"

SYNC_HOST="localhost"
SYNC_PATH="/usr/share/nginx/html/downloads/ssh_ms/"

all: lint format binaries

binaries: binary-linux binary-mac

flags:
	@echo -e "\"${LDFLAGS}\"" | sed 's/-ldflags /-ldflags "/; s/^"//'

sync:
	@rsync -rlpDvc --progress bin/{linux,darwin} "${SYNC_HOST}":"${SYNC_PATH}"

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

lint:
	@golint .

format:
	@gofmt -w .

simplify:
	@gofmt -s -w .

vet:
	@go vet -x .

clean:
	@find "${BUILD_DIR}" -type f -delete
