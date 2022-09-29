// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bhuisgen/neon/internal/app"
)

// testCommand implements the test command
type testCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

// NewTestCommand creates the command
func NewTestCommand() *testCommand {
	c := testCommand{}
	c.flagset = flag.NewFlagSet("test", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon test [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		os.Exit(2)
	}

	return &c
}

// Name returns the command name
func (c *testCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description
func (c *testCommand) Description() string {
	return "Test the configuration file"
}

// Init initializes the command
func (c *testCommand) Init(args []string) error {
	return c.flagset.Parse(args)
}

// Execute executes the command
func (c *testCommand) Execute() error {
	config, err := app.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %s\n", err)

		return err
	}

	report, err := app.TestConfig(config)
	if err != nil {
		for _, line := range report {
			fmt.Println(line)
		}
		fmt.Println("Configuration file test failed")

		return fmt.Errorf("invalid configuration")
	}

	fmt.Println("Configuration file test is successful")

	return nil
}
