package logger

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
)

// LogFileWriter
type LogFileWriter interface {
	io.Writer
	io.Closer
	Reopen() error
	Sync() error
}

// logFileWriter implements a log file writer.
type logFileWriter struct {
	name string
	flag int
	perm os.FileMode
	f    *os.File
	mu   sync.Mutex
}

// CreateLogFileWriter creates a log file writer.
func CreateLogFileWriter(name string, flag int, perm fs.FileMode) (*logFileWriter, error) {
	w := logFileWriter{
		name: name,
		flag: flag,
		perm: perm,
	}

	err := w.Reopen()
	if err != nil {
		return nil, err
	}

	return &w, nil
}

// Reopen reopens the log file.
func (w *logFileWriter) Reopen() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.f != nil {
		_ = w.f.Close()
		w.f = nil
	}

	f, err := os.OpenFile(w.name, w.flag, w.perm)
	if err != nil {
		return fmt.Errorf("open file %s: %w", w.name, err)
	}
	w.f = f

	return nil
}

// Close closes the writer.
func (w *logFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.f != nil {
		err := w.f.Close()
		w.f = nil
		if err != nil {
			return fmt.Errorf("close file %s: %w", w.name, err)
		}
	}

	return nil
}

// Write writes the given data.
func (w *logFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := w.f.Write(p)
	if err != nil {
		return n, fmt.Errorf("write file: %w", err)
	}

	return n, nil
}

// Sync commits the content.
func (w *logFileWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.f.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	return nil
}
