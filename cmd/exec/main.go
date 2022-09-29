// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
)

type command interface {
	Name() string
	Description() string
	Init(args []string) error
	Execute() error
}

// main is the entrypoint
func main() {
	commands := []command{
		NewInitCommand(),
		NewCheckCommand(),
		NewServeCommand(),
		NewTestCommand(),
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Print version information and quit")

	flag.Usage = func() {
		fmt.Println()
		fmt.Println("Usage: neon [OPTIONS] COMMAND")
		fmt.Println()
		fmt.Println("The web server ready for your Javascript application")
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
		fmt.Println()
		fmt.Println("To get more help with neon, check out our docs at https://neon.bhexpert.com/docs/latest")
		os.Exit(2)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("%s version %s, commit %s\n", name, version, commit)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
	}

	for _, c := range commands {
		if c.Name() == args[0] {
			err := c.Init(args[1:])
			if err != nil {
				os.Exit(1)
			}

			err = c.Execute()
			if err != nil {
				os.Exit(1)
			}

			os.Exit(0)
		}
	}

	flag.Usage()
}
