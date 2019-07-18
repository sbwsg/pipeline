package main

import "io"

type LogExporter interface {
	Config(map[string]string)
	Stdout() io.WriteCloser
	Stderr() io.WriteCloser
}
