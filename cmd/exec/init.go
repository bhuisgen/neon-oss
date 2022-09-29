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

// initCommand implements the init command
type initCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

// NewInitCommand creates the command
func NewInitCommand() *initCommand {
	c := initCommand{}
	c.flagset = flag.NewFlagSet("init", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon init [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		os.Exit(2)
	}

	return &c
}

// Name returns the command name
func (c *initCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description
func (c *initCommand) Description() string {
	return "Generate a new configuration file"
}

// Init initializes the command
func (c *initCommand) Init(args []string) error {
	return c.flagset.Parse(args)
}

// Execute executes the command
func (c *initCommand) Execute() error {
	err := app.GenerateConfig()
	if err != nil {
		fmt.Printf("Failed to generate configuration file: %s\n", err)

		return err
	}

	fmt.Println("Configuration file generated")

	return nil
}
