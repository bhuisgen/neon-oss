// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

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
	c.flagset.StringVar(&c.template, "template", "static", "Template name")
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
func (c *initCommand) Execute() error {
	err := neon.GenerateConfig(c.template)
	if err != nil {
		fmt.Printf("Failed to generate configuration: %s\n", err)
		return err
	}

	fmt.Println("Configuration file generated")
	return nil
}

var _ command = (*initCommand)(nil)
