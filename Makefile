# vim: ts=8:sw=8:ft=make:noai:noet
SHELL=/bin/bash

BUILD_DIR?="./bin"
GO?="`which go`"
GOLINT?="`which golint`"
RELEASE_VER?="`git rev-parse HEAD`"

XZ_COMPRESS?=1

SSH_MS_BASEPATH?=~/.ssh/cache
SSH_MS_DEFAULT_VAULT_ADDR?=https://127.0.0.1:8200
SSH_MS_DEFAULT_USERNAME?="${USER}"
SSH_MS_ID_FILE?=~/.ssh/id_rsa
SSH_MS_RENEW_THRESHOLD?=168h
SSH_MS_SECRET_MAP?="NGINX:443;PMM:8443"
SSH_MS_SECRET_PATH?=secret/ssh_ms
SSH_MS_SYNC_HOST?=localhost
SSH_MS_SYNC_PATH?=/usr/share/nginx/html/downloads/ssh_ms/
SSH_MS_USERNAME?=SSH_MS_USERNAME

DEBUG_BUILD=$(shell test "${DEBUG}" = "1" && echo 1 || echo 0)
COMPRESS_BINARY=$(shell test "${XZ_COMPRESS}" = "1" && echo 1 || echo 0)

PACKAGE=github.com/cezmunsta/ssh_ms
ifeq ($(DEBUG_BUILD), 1)
LDFLAGS=-ldflags "-X ${PACKAGE}/config.EnvBasePath=${SSH_MS_BASEPATH} -X ${PACKAGE}/cmd.Version=${RELEASE_VER} -X ${PACKAGE}/config.EnvSSHUsername=${SSH_MS_USERNAME} -X ${PACKAGE}/config.EnvSSHIdentityFile=${SSH_MS_ID_FILE} -X ${PACKAGE}/config.EnvSSHDefaultUsername=${SSH_MS_DEFAULT_USERNAME} -X ${PACKAGE}/config.EnvVaultAddr=${SSH_MS_DEFAULT_VAULT_ADDR} -X ${PACKAGE}/config.SecretPath=${SSH_MS_SECRET_PATH} -X ${PACKAGE}/vault.RenewThreshold=${SSH_MS_RENEW_THRESHOLD} -X ${PACKAGE}/config.portServiceMappings=${SSH_MS_SECRET_MAP}"
else
LDFLAGS=-ldflags "-w -X ${PACKAGE}/config.EnvBasePath=${SSH_MS_BASEPATH} -X ${PACKAGE}/cmd.Version=${RELEASE_VER} -X ${PACKAGE}/config.EnvSSHUsername=${SSH_MS_USERNAME} -X ${PACKAGE}/config.EnvSSHIdentityFile=${SSH_MS_ID_FILE} -X ${PACKAGE}/config.EnvSSHDefaultUsername=${SSH_MS_DEFAULT_USERNAME} -X ${PACKAGE}/config.EnvVaultAddr=${SSH_MS_DEFAULT_VAULT_ADDR} -X ${PACKAGE}/config.SecretPath=${SSH_MS_SECRET_PATH} -X ${PACKAGE}/vault.RenewThreshold=${SSH_MS_RENEW_THRESHOLD} -X ${PACKAGE}/config.portServiceMappings=${SSH_MS_SECRET_MAP}"
endif

VETFLAGS?=( -unusedresult -bools -copylocks -framepointer -httpresponse -json -stdmethods -printf -stringintconv -unmarshal -unsafeptr )

all: lint format test binaries

binaries: binary-linux binary-mac binary-mac-m1

flags:
	@echo -e "\"${LDFLAGS}\"" | sed 's/-ldflags /-ldflags "/; s/^"//'

sync:
	@find bin/{linux,darwin} -type f -exec chmod a+r {} \;
	@rsync -rlpDvc --progress bin/{linux,darwin} "${SSH_MS_SYNC_HOST}":"${SSH_MS_SYNC_PATH}"

binary-prep:
	@"${GO}" generate
	@mkdir -p ${BUILD_DIR}/${GOOS}/${GOARCH};

binary-mac: export GOOS=darwin
binary-mac: export GOARCH=amd64
binary-mac: export CGO_ENABLED=0
binary-mac: binary-prep
	@"${GO}" build -trimpath -o "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms" ${LDFLAGS};
ifeq ($(COMPRESS_BINARY), 1)
	@xz -fkez9 "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms";
endif

binary-mac-m1: export GOOS=darwin
binary-mac-m1: export GOARCH=arm64
binary-mac-m1: export CGO_ENABLED=0
binary-mac-m1: binary-prep
	@"${GO}" build -trimpath -o "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms" ${LDFLAGS};
ifeq ($(COMPRESS_BINARY), 1)
	@xz -fkez9 "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms";
endif

binary-linux: export GOOS=linux
binary-linux: export GOARCH=amd64
binary-linux: export CGO_ENABLED=0
binary-linux: binary-prep
	@"${GO}" build -trimpath -o "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms" ${LDFLAGS};
ifeq ($(COMPRESS_BINARY), 1)
	@xz -fkez9 "${BUILD_DIR}/${GOOS}/${GOARCH}/ssh_ms";
endif

build: binary-prep
ifeq ($(DEBUG_BUILD), 1)
	@"${GO}" build -race -trimpath -o "${BUILD_DIR}/ssh_ms.debug" ${LDFLAGS} -gcflags="all=-N -l"
else
	@"${GO}" build -race -trimpath -o "${BUILD_DIR}/ssh_ms" ${LDFLAGS}
endif

dev-vault:
	@${SHELL} podman stop dev-vault 2>/dev/null || true
	@${SHELL} scripts/dev-vault.sh 1

test:
	@"${GO}" test "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault" "${PACKAGE}/log" "${PACKAGE}/config"

mod-updates:
	@"${GO}" list -m -u all > updates.log
	@cat updates.log

lint:
	@"${GOLINT}" -set_exit_status "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault" "${PACKAGE}/log" "${PACKAGE}/config"

format: export PACKAGE=./
format:
	@"${GO}" fmt  "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault" "${PACKAGE}/log" "${PACKAGE}/config"
	@git diff --exit-code --quiet "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault" "${PACKAGE}/log" "${PACKAGE}/config"

vet:
	@"${GO}" vet "${VETFLAGS[@]}" "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault"

fix: export PACKAGE=./
fix:
	@"${GO}" tool fix -go go1.23 -diff "${PACKAGE}/ssh" "${PACKAGE}/cmd" "${PACKAGE}/vault" "${PACKAGE}/log" "${PACKAGE}/config"

clean:
	@find "${BUILD_DIR}" -type f -delete;
	@"${GO}" clean -x
	@"${GO}" clean -x -cache
	@"${GO}" clean -x -testcache
