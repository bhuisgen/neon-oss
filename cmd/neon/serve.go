package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/bhuisgen/neon/internal/app/neon"
)

// serveCommand implements the serve command.
type serveCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

// NewServeCommand creates a new serve command.
func NewServeCommand() *serveCommand {
	c := serveCommand{}
	c.flagset = flag.NewFlagSet("serve", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon serve [OPTIONS]")
		fmt.Println()
		fmt.Println("Run the server instance.")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		fmt.Println()
	}

	return &c
}

// Name returns the command name.
func (c *serveCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description.
func (c *serveCommand) Description() string {
	return "Run the server instance"
}

// Parse parses the command arguments.
func (c *serveCommand) Parse(args []string) error {
	err := c.flagset.Parse(args)
	if err != nil {
		return err
	}

	if len(c.flagset.Args()) > 0 {
		fmt.Println("The command accepts no arguments")
		return errors.New("invalid arguments")
	}

	return nil
}

// Execute executes the command.
func (c *serveCommand) Execute() error {
	config, err := neon.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %s\n", err)
		return err
	}

	err = neon.NewApplication(config).Serve()
	if err != nil {
		fmt.Printf("Failed to run instance: %s\n", err)
		return err
	}

	return nil
}

var _ command = (*serveCommand)(nil)
