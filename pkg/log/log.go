package log

import (
	"log"
	"log/slog"
	"os"
)

// ProgramLevel is the common log level.
var ProgramLevel = new(slog.LevelVar)

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func Fatal(v ...any) {
	log.Default().Print(v...)
	os.Exit(1)
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func Fatalf(format string, v ...any) {
	log.Default().Printf(format, v...)
	os.Exit(1)
}

// Fatalln is equivalent to Println() followed by a call to os.Exit(1).
func Fatalln(v ...any) {
	log.Default().Println(v...)
	os.Exit(1)
}
