package logexport

import "io"

type Exporter interface {
	io.Closer

	Config(map[string]string) error
	Stdout() io.Writer
	Stderr() io.Writer
}
