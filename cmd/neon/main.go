// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/bhuisgen/neon/internal/app/neon"
	"github.com/bhuisgen/neon/pkg/core"
)

// command
type command interface {
	Name() string
	Description() string
	Parse(args []string) error
	Execute() error
}

// main is the entrypoint.
func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

// run parses and executes the command line.
func run() error {
	commands := []command{
		NewInitCommand(),
		NewCheckCommand(),
		NewServeCommand(),
	}

	var version bool
	flag.BoolVar(&version, "v", false, "Print version information and quit")
	flag.Usage = func() {
		fmt.Println()
		fmt.Println("Usage: neon [OPTIONS] COMMAND")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Commands:")
		for _, c := range commands {
			fmt.Printf("  %-16s %s\n", c.Name(), c.Description())
		}
		fmt.Println()
		fmt.Println("Run 'neon COMMAND --help' for more information on a command.")
	}
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		return nil
	}

	if version {
		fmt.Printf("%s version %s\n", neon.Name, neon.Version)
		return nil
	}

	if v, ok := os.LookupEnv("CONFIG_FILE"); ok {
		core.CONFIG_FILE = v
	}
	if v, ok := os.LookupEnv("DEBUG"); ok {
		if v != "0" {
			core.DEBUG = true
		}
	}

	for _, c := range commands {
		if c.Name() != flag.Arg(0) {
			continue
		}
		err := c.Parse(flag.Args()[1:])
		if err != nil {
			return err
		}
		err = c.Execute()
		if err != nil {
			return err
		}
		return nil
	}

	flag.Usage()
	return errors.New("invalid command")
}
