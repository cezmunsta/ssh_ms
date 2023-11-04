//go:build ignore

package main

import (
	"fmt"
	"io/ioutil"

	"golang.org/x/mod/modfile"
)

const depVersions = `// Code generated - DO NOT EDIT
package config

const (
	vaultApiVersion = "%v"
	vaultSdkVersion = "%v"
)

`

func main() {
	var f *modfile.File
	var fb []byte
	var err error

	vApi, vSdk := "", ""

	if fb, err = ioutil.ReadFile("go.mod"); err != nil {
		panic(err)
	}

	if f, err = modfile.Parse("go.mod", fb, nil); err != nil {
		panic(err)
	}

	for _, mod := range f.Require {
		if mod.Indirect {
			continue
		}

		switch mod.Mod.Path {
		case "github.com/hashicorp/vault/api":
			vApi = mod.Mod.Version
		case "github.com/hashicorp/vault/sdk":
			vSdk = mod.Mod.Version
		}
	}

	if err := ioutil.WriteFile("config/versions.go", []byte(fmt.Sprintf(depVersions, vApi, vSdk)), 0o644); err != nil {
		panic("failed writing config/versions.go")
	}
}
