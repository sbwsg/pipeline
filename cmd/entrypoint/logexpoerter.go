package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"golang.org/x/xerrors"
)

type LogConfig struct {
	URL         string `json:"-"` // Field omitted from serialization
	Pipeline    string `json:"pipeline"`
	PipelineRun string `json:"pipelinerun"`
	Task        string `json:"task"`
	TaskRun     string `json:"taskrun"`
}

type LogMessage struct {
	LogConfig

	Stream  string `json:"stream"`
	Content string `json:"content"`
}

type HTTPJSONExporter struct {
	stdout io.WriteCloser
	stderr io.WriteCloser
}

func NewHTTPJSONExporter(config *LogConfig) (*HTTPJSONExporter, error) {
	if config.URL == "" {
		return nil, errors.New("error creating http json log exporter: no logging url provided")
	}
	if config.Task == "" {
		return nil, errors.New("error creating http json log exporter: no task name provided")
	}
	h := &HTTPJSONExporter{}
	h.stdout = newHTTPWriter("stdout", config)
	h.stderr = newHTTPWriter("stderr", config)
	log.Printf("logging config: %+v", config)
	return h, nil
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

// httpWriter POSTs lists of log lines to a configured HTTP url. It implements the
// WriteCloser interface. Writes are asynchronous - they are initially buffered so as
// not to block the writer.
type httpWriter struct {
	stream string
	config *LogConfig

	messageQueue chan *LogMessage
	errMu        sync.Mutex
	sendErr      error
	stopCh       chan struct{}
}

func newHTTPWriter(stream string, config *LogConfig) *httpWriter {
	w := &httpWriter{
		stream:       stream,
		config:       config,
		messageQueue: make(chan *LogMessage),
		stopCh:       make(chan struct{}),
	}
	w.startSendLoop()
	return w
}

// Write buffers a log line to be sent to the logging destination. Sending is
// asynchronous and therefore the error returned may stem from a previous Write.
func (w *httpWriter) Write(line []byte) (int, error) {
	w.messageQueue <- &LogMessage{
		LogConfig: *w.config,
		Stream:    w.stream,
		Content:   string(line),
	}
	if w.sendErr != nil {
		w.errMu.Lock()
		err := w.sendErr
		w.sendErr = nil
		w.errMu.Unlock()
		return len(line), err
	}
	return len(line), nil
}

func (w *httpWriter) Close() error {
	close(w.stopCh)
	var err error
	if w.sendErr != nil {
		w.errMu.Lock()
		err = w.sendErr
		w.errMu.Unlock()
	}
	return err
}

// startSendLoop launches a go routine to buffer new log lines and another
// to POST them out.
func (w *httpWriter) startSendLoop() {
	var logBuf []*LogMessage
	var mu sync.Mutex
	go func() {
		for {
			select {
			case msg := <-w.messageQueue:
				mu.Lock()
				logBuf = append(logBuf, msg)
				mu.Unlock()
			case <-w.stopCh:
				return
			}
		}
	}()
	go func() {
		var payload []*LogMessage
		for {
			select {
			case <-w.stopCh:
				return
			default:
				if logBuf != nil {
					mu.Lock()
					payload = logBuf
					logBuf = nil
					mu.Unlock()

					err := w.sendLogs(payload)

					if err != nil && w.sendErr == nil {
						w.errMu.Lock()
						w.sendErr = err
						w.errMu.Unlock()
					}
				}
			}
		}
	}()
}

// sendLogs serializes and POSTs a slice of log lines to w's configured destination url.
func (w *httpWriter) sendLogs(payload []*LogMessage) error {
	pr, pw := io.Pipe()
	var jsonErr error
	go func() {
		jsonErr = json.NewEncoder(pw).Encode(payload)
		pw.Close()
	}()
	if resp, err := http.Post(w.config.URL, "application/json", pr); err != nil {
		return xerrors.Errorf("error posting log line: %w", err)
	} else {
		// Drain any response body to let Transport reuse connection
		// See Body field's comment (from https://golang.org/pkg/net/http/#Response):
		// The default HTTP client's Transport may not reuse HTTP/1.x "keep-alive"
		// TCP connections if the Body is not read to completion and closed.
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
	return jsonErr
}
