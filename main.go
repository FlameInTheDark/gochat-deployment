package main

import (
	"context"
	"fmt"
	"os"

	"github.com/FlameInTheDark/gochat-deployment/deployer"
)

func main() {
	if err := deployer.Run(context.Background(), deploymentBundle, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
