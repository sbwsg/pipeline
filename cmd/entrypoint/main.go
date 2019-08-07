/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/tektoncd/pipeline/pkg/entrypoint"
)

var (
	ep              = flag.String("entrypoint", "", "Original specified entrypoint to execute")
	waitFile        = flag.String("wait_file", "", "If specified, file to wait for")
	waitFileContent = flag.Bool("wait_file_content", false, "If specified, expect wait_file to have content")
	postFile        = flag.String("post_file", "", "If specified, file to write upon completion")

	logURL      = flag.String("log_url", "", "If specified, the HTTP endpoint to send log output")
	pipeline    = flag.String("pipeline", "", "If specified, the pipeline this entrypoint is being executed by")
	pipelinerun = flag.String("pipelinerun", "", "If specified, the pipelinerun this entrypoint is being executed by")
	task        = flag.String("task", "", "If specified, the task this entrypoint is being executed by")
	taskrun     = flag.String("taskrun", "", "If specified, the taskrun this entrypoint is being executed by")

	waitPollingInterval = time.Second
)

func main() {
	flag.Parse()

	f, err := os.Create("./cpuprofile")
	if err != nil {
		log.Printf("error opening cpuprofile for writing: %v", err)
		os.Exit(21)
	}
	log.Printf("STARTING CPU PROFILE")
	pprof.StartCPUProfile(f)

	exitCh := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Printf("STOPPING CPU PROFILE")
			pprof.StopCPUProfile()
			log.Printf("STOPPED CPU PROFILE")
			close(exitCh)
		}
	}()

	var runner *RealRunner
	if *logURL != "" {
		runner = NewRealRunner(&LogConfig{
			URL:         *logURL,
			Pipeline:    *pipeline,
			PipelineRun: *pipelinerun,
			Task:        *task,
			TaskRun:     *taskrun,
		})
	} else {
		runner = NewRealRunner(nil)
	}
	e := entrypoint.Entrypointer{
		Entrypoint:      *ep,
		WaitFile:        *waitFile,
		WaitFileContent: *waitFileContent,
		PostFile:        *postFile,
		Args:            flag.Args(),
		Waiter:          &RealWaiter{},
		Runner:          runner,
		PostWriter:      &RealPostWriter{},
	}
	if err := e.Go(); err != nil {
		<-exitCh
		switch err.(type) {
		case skipError:
			os.Exit(0)
		case *exec.ExitError:
			// Copied from https://stackoverflow.com/questions/10385551/get-exit-code-go
			// This works on both Unix and Windows. Although
			// package syscall is generally platform dependent,
			// WaitStatus is defined for both Unix and Windows and
			// in both cases has an ExitStatus() method with the
			// same signature.
			if status, ok := err.(*exec.ExitError).Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
			log.Fatalf("Error executing command (ExitError): %v", err)
		default:
			log.Fatalf("Error executing command: %v", err)
		}
	}
	<-exitCh
}
