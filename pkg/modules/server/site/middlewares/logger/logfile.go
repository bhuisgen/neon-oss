package logger

import (
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

// Reopen reopens the file writer.
func (w *logFileWriter) Reopen() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.f != nil {
		w.f.Close()
		w.f = nil
	}

	f, err := os.OpenFile(w.name, w.flag, w.perm)
	if err != nil {
		return err
	}
	w.f = f

	return nil
}

// Close closes the file writer.
func (w *logFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.f != nil {
		err := w.f.Close()
		w.f = nil
		if err != nil {
			return err
		}
	}

	return nil
}

// Write writes the given data to the file writer.
func (w *logFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.f.Write(p)
}

// Sync commits the log file writer.
func (w *logFileWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.f.Sync()
}
