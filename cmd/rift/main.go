// Command rift is the Rift remote Docker context and artifact cache server.
// It is deployed as a standalone VM by MotherGoose via OpenTofu and provides
// shared Docker image caching and remote context for CI runners.
package main

import (
	"fmt"
	"os"

	"github.com/polar-gosling/gosling/internal/cli"
)

func main() {
	if err := cli.ExecuteRift(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
