package apiscope

import (
	"fmt"
	"io"
	"runtime/debug"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/logging"
	"github.com/phergul/apiscope/internal/persist"
	"github.com/phergul/apiscope/internal/spec"
	"github.com/phergul/apiscope/internal/tui"
)

type runner interface {
	Run() error
}

var newProgram = func(service *app.Service, source string, input io.Reader, output io.Writer) runner {
	return tui.NewProgram(service, source, input, output)
}

var newDiagnosticsLogger = logging.NewDefaultLogger
var readBuildInfo = debug.ReadBuildInfo

// Version is the CLI version string. It can be overridden at build time via ldflags.
var Version = "dev"

func resolvedVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if buildInfo, ok := readBuildInfo(); ok {
		version := buildInfo.Main.Version
		if version != "" && version != "(devel)" {
			return version
		}
	}
	return "dev"
}

func Run(args []string, input io.Reader, output, errOutput io.Writer) int {
	for _, arg := range args {
		if arg == "--version" {
			fmt.Fprintln(output, resolvedVersion())
			return 0
		}
	}

	if len(args) != 1 {
		fmt.Fprintln(errOutput, "usage: apiscope <spec-source>")
		return 2
	}

	logger, closer, err := newDiagnosticsLogger()
	if err != nil {
		fmt.Fprintf(errOutput, "apiscope: diagnostics logging disabled: %v\n", err)
		logger = logging.NopLogger()
	} else if closer != nil {
		defer closer.Close()
	}

	service := app.NewService(spec.NewLoader(nil, logger), nil, persist.NewStore(""), logger)
	program := newProgram(service, args[0], input, output)
	if err := program.Run(); err != nil {
		fmt.Fprintf(errOutput, "apiscope: %v\n", err)
		return 1
	}

	return 0
}
