package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/bhuisgen/neon/internal/app/neon"
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
	if err := run(); err != nil {
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

	if version {
		fmt.Printf("%s version %s\n", neon.Name, neon.Version)
		return nil
	}

	if len(flag.Args()) == 0 {
		flag.Usage()
		return nil
	}

	for _, c := range commands {
		if c.Name() != flag.Arg(0) {
			continue
		}
		if err := c.Parse(flag.Args()[1:]); err != nil {
			flag.Usage()
			return errors.New("parse command")
		}
		if err := c.Execute(); err != nil {
			return errors.New("execute command")
		}
		return nil
	}

	flag.Usage()
	return errors.New("invalid command")
}
