// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"flag"
	"fmt"
	"runtime"

	"github.com/bhuisgen/neon/internal/app"
)

// versionCommand implements the version command
type versionCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

// NewVersionCommand creates a command
func NewVersionCommand() *versionCommand {
	c := versionCommand{}
	c.flagset = flag.NewFlagSet("version", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon version [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
	}

	return &c
}

// Name returns the command name
func (c *versionCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description
func (c *versionCommand) Description() string {
	return "Show the version information"
}

// Init initializes the command
func (c *versionCommand) Init(args []string) error {
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
func (c *versionCommand) Execute() error {
	fmt.Printf("%s version %s\n", app.Name, app.Version)
	fmt.Println()
	fmt.Printf("API version: %s\n", app.API)
	if app.Commit != "" {
		fmt.Printf("Git commit: %s\n", app.Commit)
	}
	if app.Date != "" {
		fmt.Printf("Built: %s\n", app.Date)
	}
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	return nil
}
