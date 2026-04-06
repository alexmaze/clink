package cli

import (
	"errors"
	"fmt"
	"io"
)

var Version = "dev"

var errSilentExit = errors.New("silent exit")

type exitError struct {
	err    error
	silent bool
}

func (e *exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	return e.err
}

func IsSilent(err error) bool {
	var target *exitError
	return errors.As(err, &target) && target.silent
}

type app struct {
	stdout io.Writer
	stderr io.Writer
}

func Run(args []string, stdout, stderr io.Writer) error {
	return app{stdout: stdout, stderr: stderr}.run(args)
}

func (a app) run(args []string) error {
	if len(args) < 2 {
		a.printUsage()
		return &exitError{err: errSilentExit, silent: true}
	}

	switch args[1] {
	case "apply":
		return a.runApply(parseApply(args[2:]))
	case "check":
		return a.runCheck(parseCheck(args[2:]))
	case "restore":
		return a.runRestore(parseRestore(args[2:]))
	case "add":
		return a.runAdd(parseAdd(args[2:]))
	case "version", "--version", "-v":
		_, err := fmt.Fprintln(a.stdout, Version)
		return err
	default:
		a.printUsage()
		return &exitError{err: errSilentExit, silent: true}
	}
}
