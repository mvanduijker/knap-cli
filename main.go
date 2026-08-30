package main

import (
	"os"

	"github.com/mvanduijker/knap-cli/cmd"
)

// version is set by goreleaser at build time.
var version = "dev"

func main() {
	os.Exit(cmd.Execute(version))
}
