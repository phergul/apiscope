package main

import (
	"os"

	apiscope "github.com/phergul/apiscope/cmd/apiscope"
)

func main() {
	os.Exit(apiscope.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
