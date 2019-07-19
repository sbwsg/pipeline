package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/tektoncd/pipeline/cmd/entrypoint/logexport"
	"github.com/tektoncd/pipeline/pkg/entrypoint"
)

// TODO(jasonhall): Test that original exit code is propagated and that
// stdout/stderr are collected -- needs e2e tests.

// RealRunner actually runs commands.
type RealRunner struct {
	logs logexport.Exporter
}

var _ entrypoint.Runner = (*RealRunner)(nil)

func NewRealRunner() (*RealRunner, error) {
	logsconfig := map[string]string{
		"destination": "http://localhost:9999",
	}
	logs := &logexport.HTTPJSONExporter{}
	if err := logs.Config(logsconfig); err != nil {
		return nil, err
	}
	return &RealRunner{
		logs: logs,
	}, nil
}

func (rr *RealRunner) Run(args ...string) error {
	if len(args) == 0 {
		return nil
	}
	name, args := args[0], args[1:]

	cmd := exec.Command(name, args...)

	if rr.logs != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, rr.logs.Stdout())
		cmd.Stderr = io.MultiWriter(os.Stderr, rr.logs.Stderr())
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
