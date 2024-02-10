package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/bhuisgen/neon/internal/app/neon"
)

// checkCommand implements the check command.
type checkCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

// NewCheckCommand creates a new check command.
func NewCheckCommand() *checkCommand {
	c := checkCommand{}
	c.flagset = flag.NewFlagSet("check", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon check [OPTIONS]")
		fmt.Println()
		fmt.Println("Check the configuration.")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		fmt.Println()
	}

	return &c
}

// Name returns the command name.
func (c *checkCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description.
func (c *checkCommand) Description() string {
	return "Check the configuration"
}

// Parse parses the command arguments.
func (c *checkCommand) Parse(args []string) error {
	if err := c.flagset.Parse(args); err != nil {
		return errors.New("parse arguments")
	}
	if len(c.flagset.Args()) > 0 {
		return errors.New("check arguments")
	}
	return nil
}

// Execute executes the command.
func (c *checkCommand) Execute() error {
	config, err := neon.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return fmt.Errorf("load config: %v", err)
	}

	if err := neon.New(config).Check(); err != nil {
		fmt.Println("Configuration is not valid")
		return fmt.Errorf("check: %v", err)
	}

	fmt.Println("Configuration is valid")

	return nil
}

var _ command = (*checkCommand)(nil)
