package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/xerrors"
)

const (
	SendBufferSize = 1000
	MaxErrors      = 100
)

// LogConfig provides fields to config the behaviour of the HTTPJSONExporter.
type LogConfig struct {
	// URL is the destination HTTP address for log messages to be sent to.
	URL string `json:"-"` // Field omitted from serialization
	// Pipeline is the name of the pipeline that this entrypoint is executing as part of.
	Pipeline string `json:"pipeline"`
	// PipelineRun is the name of the pipelinerun that this entrypoint is executing as part of.
	PipelineRun string `json:"pipelinerun"`
	// Task is the name of the task that this entrypoint is executing as part of.
	Task string `json:"task"`
	// TaskRun is the name of the taskrun that this entrypoint is executing as part of.
	TaskRun string `json:"taskrun"`
}

// LogMessage is the format of log entries POSTed by the HTTPJSONExporter.
type LogMessage struct {
	LogConfig

	Stream       string `json:"stream"`
	Content      string `json:"content"`
	ContentBytes []byte `json:"-"` // Field omitted from serialization
}

// HTTPJSONExporter accepts Writes from either stdout or stderr and POSTs them
// to a configured HTTP endpoint.
type HTTPJSONExporter struct {
	stdout io.WriteCloser
	stderr io.WriteCloser
}

// NewHTTPJSONExporter creates an HTTPJSONExporter. It expects a destination URL and, at minimum,
// a Task name to be provided through the passed-in LogConfig struct.
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
	return h, nil
}

// Stderr returns a Writer for processing and sending out log messages from
// a stdout stream.
func (h *HTTPJSONExporter) Stdout() io.Writer {
	return h.stdout
}

// Stderr returns a Writer for processing and sending out log messages from
// a stderr stream.
func (h *HTTPJSONExporter) Stderr() io.Writer {
	return h.stderr
}

// Close stops the HTTPJSONExporter from processing any more Write calls.
func (h *HTTPJSONExporter) Close() error {
	stdoutErr := h.stdout.Close()
	stderrErr := h.stderr.Close()
	if stdoutErr != nil {
		return stdoutErr
	}
	return stderrErr
}

// httpWriter POSTs lists of log lines to a configured HTTP url. It implements the
// WriteCloser interface. Writes are asynchronous, buffered so as not to block the writer.
type httpWriter struct {
	stream string
	config *LogConfig

	messageQueue chan *LogMessage
	errCh        chan error
	stopCh       chan struct{}
}

func newHTTPWriter(stream string, config *LogConfig) *httpWriter {
	w := &httpWriter{
		stream:       stream,
		config:       config,
		messageQueue: make(chan *LogMessage),
		stopCh:       make(chan struct{}),
		errCh:        make(chan error, MaxErrors),
	}
	w.startSendLoop()
	return w
}

// Write buffers a log line to be sent to the logging destination. Sending is
// asynchronous and therefore the error returned may stem from a previous Write.
func (w *httpWriter) Write(line []byte) (int, error) {
	w.messageQueue <- &LogMessage{
		LogConfig:    *w.config,
		Stream:       w.stream,
		ContentBytes: line,
	}
	select {
	case err := <-w.errCh:
		return len(line), err
	default:
		return len(line), nil
	}
}

// Close stops any more writes from being sent out by w
func (w *httpWriter) Close() error {
	close(w.stopCh)
	select {
	case err := <-w.errCh:
		return err
	default:
		return nil
	}
}

// startSendLoop launches a go routine to buffer new log lines and uses another
// to send them when the buffer has reached SendBufferSize. When w's stopCh is
// closed one final send is performed to try and flush any remaining messages.
func (w *httpWriter) startSendLoop() {
	go func() {
		var logBuf []*LogMessage
		bufSize := 0
		for {
			select {
			case <-w.stopCh:
				break
			case msg := <-w.messageQueue:
				logBuf = append(logBuf, msg)
				bufSize += len(msg.ContentBytes)
				if bufSize >= SendBufferSize {
					payload := logBuf
					logBuf = nil
					bufSize = 0
					w.sendLogsNonBlocking(payload)
				}
			}
		}
		if bufSize > 0 {
			w.sendLogs(logBuf)
		}
	}()
}

// sendLogs serializes and POSTs a slice of log lines to w's configured destination url.
func (w *httpWriter) sendLogs(payload []*LogMessage) error {
	for i := range payload {
		payload[i].Content = string(payload[i].ContentBytes)
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return xerrors.Errorf("error marshalling log lines to json: %w", err)
	}
	if resp, err := http.Post(w.config.URL, "application/json", bytes.NewBuffer(b)); err != nil {
		return xerrors.Errorf("error sending log lines: %w", err)
	} else {
		// Drain any response body to let Transport reuse connection
		// See Body field's comment (from https://golang.org/pkg/net/http/#Response):
		// The default HTTP client's Transport may not reuse HTTP/1.x "keep-alive"
		// TCP connections if the Body is not read to completion and closed.
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
	return nil
}

// sendLogsNonBlocking wraps a call to sendLogs() in a go routine and emits
// any errors through w's errCh. If too many errors have accumulated on the
// errCh then w is shut down.
func (w *httpWriter) sendLogsNonBlocking(payload []*LogMessage) {
	go func() {
		if err := w.sendLogs(payload); err != nil {
			select {
			case w.errCh <- err:
				// ok
			default:
				// Reached MaxErrors. Abandon ship!
				close(w.stopCh)
				return
			}
		}
	}()
}
