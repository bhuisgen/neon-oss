package main

import (
	"errors"
	"flag"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// versionCommand implements the version command.
type versionCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

var (
	Name    string = "Neon"
	Version string = "dev"
	Commit  string = "-"
	Date    string = "-"
)

// NewVersionCommand creates a new version command.
func NewVersionCommand() *versionCommand {
	c := versionCommand{}
	c.flagset = flag.NewFlagSet("version", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon version [OPTIONS]")
		fmt.Println()
		fmt.Println("Show the Neon version information.")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		fmt.Println()
	}

	return &c
}

// Name returns the command name.
func (c *versionCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description.
func (c *versionCommand) Description() string {
	return "Show version information"
}

// Parse parses the command arguments.
func (c *versionCommand) Parse(args []string) error {
	if err := c.flagset.Parse(args); err != nil {
		return errors.New("parse arguments")
	}
	if len(c.flagset.Args()) > 0 {
		return errors.New("check arguments")
	}
	return nil
}

// Execute executes the command.
func (c *versionCommand) Execute() error {
	fmt.Printf("%s\n", Name)
	fmt.Printf(" %-19s%s\n", "Version:", Version)
	fmt.Printf(" %-19s%s\n", "Commit:", Commit)
	fmt.Printf(" %-19s%s\n", "Built:", Date)
	fmt.Printf(" %-19s%s\n", "OS/Arch:", strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "/"))
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf(" %-19s%s\n", "Go version:", buildInfo.GoVersion)
	}

	return nil
}

var _ command = (*versionCommand)(nil)
