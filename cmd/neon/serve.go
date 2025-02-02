package main

import (
	"context"
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
	if err := c.flagset.Parse(args); err != nil {
		return errors.New("parse arguments")
	}
	if len(c.flagset.Args()) > 0 {
		return errors.New("check arguments")
	}
	return nil
}

// Execute executes the command.
func (c *serveCommand) Execute() error {
	fmt.Println()
	fmt.Println(Name, Version)
	fmt.Println("Copyright (C) 2022-2025 Boris HUISGEN")
	fmt.Println("All rights reserved.")
	fmt.Println()

	config, err := neon.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return fmt.Errorf("load config: %v", err)
	}

	if err := neon.New(config).Serve(context.Background()); err != nil {
		fmt.Printf("Failed to serve: %s\n", err)
		return fmt.Errorf("serve: %v", err)
	}

	return nil
}

var _ command = (*serveCommand)(nil)
