package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/bhuisgen/neon/internal/app/neon"
)

// initCommand implements the init command.
type initCommand struct {
	flagset  *flag.FlagSet
	template string
	verbose  bool
}

// NewInitCommand creates a new init command.
func NewInitCommand() *initCommand {
	c := initCommand{}
	c.flagset = flag.NewFlagSet("init", flag.ExitOnError)
	c.flagset.StringVar(&c.template, "t", "default", "Template name (default,example)")
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon init [OPTIONS]")
		fmt.Println()
		fmt.Println("Generate a new configuration file.")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		fmt.Println()
	}

	return &c
}

// Name returns the command name.
func (c *initCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description.
func (c *initCommand) Description() string {
	return "Generate a new configuration file"
}

// Parse parses the command arguments.
func (c *initCommand) Parse(args []string) error {
	if err := c.flagset.Parse(args); err != nil {
		return errors.New("parse arguments")
	}
	if len(c.flagset.Args()) > 0 {
		return errors.New("check arguments")
	}
	return nil
}

// Execute executes the command.
func (c *initCommand) Execute() error {
	if err := neon.GenerateConfig(c.template); err != nil {
		return fmt.Errorf("generate config: %v", err)
	}

	return nil
}

var _ command = (*initCommand)(nil)
