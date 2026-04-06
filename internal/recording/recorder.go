package recording

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Header is the asciicast v2 header.
type Header struct {
	Version   int               `json:"version"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Env       map[string]string `json:"env,omitempty"`
}

// Recorder writes terminal output in asciicast v2 format.
type Recorder struct {
	path      string
	width     int
	height    int
	file      *os.File
	startTime time.Time
}

// New creates a recorder targeting the given path.
func New(path string, width, height int) *Recorder {
	return &Recorder{
		path:   path,
		width:  width,
		height: height,
	}
}

// WriteHeader writes the asciicast v2 header to the file.
func (r *Recorder) WriteHeader() error {
	f, err := os.Create(r.path)
	if err != nil {
		return fmt.Errorf("creating recording file: %w", err)
	}
	r.file = f
	r.startTime = time.Now()

	header := Header{
		Version:   2,
		Width:     r.width,
		Height:    r.height,
		Timestamp: r.startTime.Unix(),
		Env: map[string]string{
			"TERM":  "xterm-256color",
			"SHELL": "/bin/zsh",
		},
	}

	data, err := json.Marshal(header)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(r.file, "%s\n", data)
	return err
}

// Close flushes and closes the recording file.
func (r *Recorder) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// Path returns the recording file path.
func (r *Recorder) Path() string {
	return r.path
}
