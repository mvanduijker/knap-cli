package main

import (
	"os"

	"github.com/mvanduijker/knap-cli/internal/cli"
)

// version is set by goreleaser at build time.
var version = "dev"

func main() {
	os.Exit(cli.Execute(version))
}
