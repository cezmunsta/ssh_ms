//go:build ignore
package main

import (
    "fmt"
    "os"
	"runtime/debug"
    "strings"

    "github.com/hashicorp/vault/api"
    "github.com/hashicorp/vault/sdk/helper/testcluster"
)

const depVersions = `// DO NOT EDIT - Code generated by generate_versions.go
package config

const (
	vaultAPIVersion = "%v"
	vaultSDKVersion = "%v"
)
`


func main() {
	vApi, vSdk := "", ""

    if true == false {
        fmt.Printf("%v", api.EnvVaultAddress)
        fmt.Printf("%v", testcluster.EnvVaultLicenseCI)
    }

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
            if !strings.Contains(dep.Path, "github.com/hashicorp/vault") {
                continue
            }
			if strings.Contains(dep.Path, "github.com/hashicorp/vault/sdk") {
				vSdk = dep.Version
			} else if strings.Contains(dep.Path, "github.com/hashicorp/vault/api") {
                vApi = dep.Version
            }
		}
	}

	if err := os.WriteFile("config/versions.go", []byte(fmt.Sprintf(depVersions, vApi, vSdk)), 0o644); err != nil {
		panic("failed writing config/versions.go")
	}
}
