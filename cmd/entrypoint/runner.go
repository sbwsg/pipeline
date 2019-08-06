package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/tektoncd/pipeline/pkg/entrypoint"
	"golang.org/x/xerrors"
)

// TODO(jasonhall): Test that original exit code is propagated and that
// stdout/stderr are collected -- needs e2e tests.

// RealRunner actually runs commands.
type RealRunner struct {
	logConfig *LogConfig
}

var _ entrypoint.Runner = (*RealRunner)(nil)

func NewRealRunner(logConfig *LogConfig) *RealRunner {
	return &RealRunner{
		logConfig: logConfig,
	}
}

func (rr *RealRunner) Run(args ...string) error {
	if len(args) == 0 {
		return nil
	}
	name, args := args[0], args[1:]

	cmd := exec.Command(name, args...)

	if rr.logConfig != nil {
		logexporter, err := NewHTTPJSONExporter(rr.logConfig)
		if err != nil {
			return xerrors.Errorf("unable to construct log exporter: %w", err)
		}
		cmd.Stdout = io.MultiWriter(os.Stdout, logexporter.Stdout())
		cmd.Stderr = io.MultiWriter(os.Stderr, logexporter.Stderr())
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
