// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/bhuisgen/neon/internal/app"
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
		NewCheckCommand(),
		NewInitCommand(),
		NewServeCommand(),
		NewVersionCommand(),
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "Print version information and quit")

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
		os.Exit(0)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("%s version %s\n", app.Name, app.Version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
	}

	if v, ok := os.LookupEnv("DEBUG"); ok {
		if v != "0" {
			app.DEBUG = true
		}
	}
	if v, ok := os.LookupEnv("CONFIG_FILE"); ok {
		s, err := os.Stat(v)
		if err != nil {
			fmt.Printf("Invalid value for environment variable CONFIG_FILE: %s\n", err)
			os.Exit(1)
		}
		if s.IsDir() {
			fmt.Printf("Invalid value for environment variable CONFIG_FILE: file is a directory\n")
			os.Exit(1)
		}
		app.CONFIG_FILE = v
	}
	if v, ok := os.LookupEnv("LISTEN_PORT"); ok {
		port, err := strconv.ParseInt(v, 10, 0)
		if err != nil {
			fmt.Printf("Invalid value for environment variable LISTEN_PORT: %s\n", err)
			os.Exit(1)
		}
		app.LISTEN_PORT = int(port)
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
