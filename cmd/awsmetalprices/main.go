package main

import (
	"os"

	"github.com/sjramblings/awsmetalprices/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
