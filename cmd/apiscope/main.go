package apiscope

import (
	"fmt"
	"io"

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

func Run(args []string, input io.Reader, output, errOutput io.Writer) int {
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
