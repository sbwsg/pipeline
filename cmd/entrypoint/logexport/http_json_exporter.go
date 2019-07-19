package logexport

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/xerrors"
)

var (
	ErrDestinationNotSet = errors.New("http log exporter: destination has not been set, nowhere to send log messages")
)

type LogLine struct {
	Stream  string `json:"stream"`
	Content string `json:"content"`
}

type LogMeta struct {
	Pipeline    string `json:"pipeline"`
	PipelineRun string `json:"pipelinerun"`
	Task        string `json:"task"`
	TaskRun     string `json:"taskrun"`
}

type SingleLineMessage struct {
	LogLine
	LogMeta
}

// HTTPJSONExporter exports log lines to a user-configured HTTP endpoint. Lines
// are buffered to reduce the number of individual HTTP requests. Messages received
// from this exporter may include multiple log lines and those lines may come from
// different streams. The stream name is part of the received JSON message.
type HTTPJSONExporter struct {
	destination *url.URL
	stdout      io.WriteCloser
	stderr      io.WriteCloser
}

var _ Exporter = (*HTTPJSONExporter)(nil)

func (h *HTTPJSONExporter) Config(conf map[string]string) error {
	if destination, has := conf["destination"]; !has {
		return ErrDestinationNotSet
	} else {
		var err error
		h.destination, err = url.Parse(destination)
		if err != nil {
			return xerrors.Errorf("http log exporter: unable to parse destination url: %w", err)
		}
	}
	meta := LogMeta{
		Pipeline:    conf["pipeline"],
		PipelineRun: conf["pipelinerun"],
		Task:        conf["task"],
		TaskRun:     conf["taskrun"],
	}
	h.stdout = &httpWriter{meta, "stdout", h.destination}
	h.stderr = &httpWriter{meta, "stderr", h.destination}
	return nil
}

func (h *HTTPJSONExporter) Stdout() io.Writer {
	return h.stdout
}

func (h *HTTPJSONExporter) Stderr() io.Writer {
	return h.stderr
}

func (h *HTTPJSONExporter) Close() error {
	stdoutErr := h.stdout.Close()
	stderrErr := h.stderr.Close()
	if stdoutErr != nil {
		return stdoutErr
	}
	return stderrErr
}

type httpWriter struct {
	LogMeta

	stream      string
	destination *url.URL
}

var _ io.WriteCloser = (*httpWriter)(nil)

// May want to make this async / non-blocking so that the io.Multiwriter
// in RealRunner doesn't have to hang around waiting for this func's
// http.Post to complete before another log line can be received.
func (h *httpWriter) Write(line []byte) (int, error) {
	msg := &SingleLineMessage{
		LogMeta: h.LogMeta,
		LogLine: LogLine{
			Stream:  h.stream,
			Content: string(line),
		},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	if _, err := http.Post(h.destination.String(), "text/plain", bytes.NewReader(b)); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (h *httpWriter) Close() error {
	// no-op when nothing is buffered & needs flushing
	// will need to flush if switching to buffered impl (example: google storage api defaults to
	// 256k chunk size for a single request. consider buffering that many log lines if it helps perf to
	// reduce http fluff)
	return nil
}
