package main

import (
	"errors"
	"io"
	"net/url"
)

type HTTPJSONLogExporter struct {
	Error       error
	destination url.URL
	stdout      io.WriteCloser
	stderr      io.WriteCloser
}

const ErrDestinationNotSet = errors.New("log exporter destination has not been configured")

func (h *HTTPJSONLogExporter) Config(conf map[string]string) {
	if destination, has := conf["destination"]; !has {
		h.Error = ErrDestinationNotSet
	} else {
		h.destination = destination
	}
}

func (h *HTTPJSONLogExporter) Stdout() io.WriteCloser {
	return h.stdout
}

func (h *HTTPJSONLogExporter) Stderr() io.WriteCloser {
	return h.stderr
}
