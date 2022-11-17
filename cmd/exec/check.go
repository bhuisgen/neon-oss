// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/bhuisgen/neon/internal/app"
)

// checkCommand implements the check command
type checkCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

// NewCheckCommand creates the command
func NewCheckCommand() *checkCommand {
	c := checkCommand{}
	c.flagset = flag.NewFlagSet("check", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon check [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		os.Exit(0)
	}

	return &c
}

// Name returns the command name
func (c *checkCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description
func (c *checkCommand) Description() string {
	return "Check the configuration file"
}

// Init initializes the command
func (c *checkCommand) Init(args []string) error {
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

// Execute executes the command
func (c *checkCommand) Execute() error {
	config, err := app.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %s\n", err)

		return err
	}

	report, err := app.CheckConfig(config)
	if err != nil {
		for _, line := range report {
			fmt.Println(line)
		}
		fmt.Println("Configuration is not valid")

		return fmt.Errorf("check failure")
	}

	fmt.Println("Configuration is valid")

	return nil
}
