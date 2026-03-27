package main

import (
	"fmt"
	"io"
	"os"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/spec"
	"github.com/phergul/apiscope/internal/tui"
)

type runner interface {
	Run() error
}

var newProgram = func(service *app.Service, source string, input io.Reader, output io.Writer) runner {
	return tui.NewProgram(service, source, input, output)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, input io.Reader, output, errOutput io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOutput, "usage: apiscope <spec-source>")
		return 2
	}

	service := app.NewService(spec.NewLoader(nil))
	program := newProgram(service, args[0], input, output)
	if err := program.Run(); err != nil {
		fmt.Fprintf(errOutput, "apiscope: %v\n", err)
		return 1
	}

	return 0
}
