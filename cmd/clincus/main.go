package main

import (
	"context"
	"os"

	"github.com/bketelsen/clincus/internal/cli"
)

func main() {
	if err := cli.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
