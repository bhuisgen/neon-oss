// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bhuisgen/neon/internal/app"
)

// serveCommand implements the serve command
type serveCommand struct {
	flagset *flag.FlagSet
	verbose bool
}

// NewServeCommand creates the command
func NewServeCommand() *serveCommand {
	c := serveCommand{}
	c.flagset = flag.NewFlagSet("serve", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon serve [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		os.Exit(2)
	}

	return &c
}

// Name returns the command name
func (c *serveCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description
func (c *serveCommand) Description() string {
	return "Run the server instance"
}

// Init initializes the command
func (c *serveCommand) Init(args []string) error {
	return c.flagset.Parse(args)
}

// Execute executes the command
func (c *serveCommand) Execute() error {
	config, err := app.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %s\n", err)

		return err
	}

	_, err = app.TestConfig(config)
	if err != nil {
		fmt.Printf("Failed to validate configuration: %s\n", err)

		return err
	}

	var servers []*app.Server

	fetcher := app.NewFetcher(config.Fetcher)
	loader := app.NewLoader(config.Loader, fetcher)

	for _, configServer := range config.Server {
		var renderers []app.Renderer

		if configServer.Rewrite.Enable {
			rewrite, err := app.CreateRewriteRenderer(&configServer.Rewrite)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, rewrite)
		}

		if configServer.Header.Enable {
			header, err := app.CreateHeaderRenderer(&configServer.Header)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}

			renderers = append(renderers, header)
		}

		if configServer.Static.Enable {
			static, err := app.CreateStaticRenderer(&configServer.Static)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, static)
		}

		if configServer.Robots.Enable {
			robots, err := app.CreateRobotsRenderer(&configServer.Robots, loader)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, robots)
		}

		if configServer.Sitemap.Enable {
			sitemap, err := app.CreateSitemapRenderer(&configServer.Sitemap, fetcher)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, sitemap)
		}

		if configServer.Index.Enable {
			index, err := app.CreateIndexRenderer(&configServer.Index, fetcher)
			if err != nil {
				fmt.Printf("Failed to create index renderer: %s\n", err)
				return err
			}
			renderers = append(renderers, index)
		}

		if configServer.Default.Enable {
			d, err := app.CreateDefaultRenderer(&configServer.Default)
			if err != nil {
				fmt.Printf("Failed to create default renderer: %s\n", err)
				return err
			}
			renderers = append(renderers, d)
		}

		e, err := app.CreateErrorRenderer(&app.ErrorRendererConfig{})
		if err != nil {
			fmt.Printf("Failed to create error renderer: %s\n", err)
			return err
		}
		renderers = append(renderers, e)

		server, err := app.CreateServer(configServer, renderers...)
		if err != nil {
			fmt.Printf("Failed to create server: %s\n", err)
			return err
		}

		servers = append(servers, server)
	}

	if _, ok := os.LookupEnv("DEBUG"); ok {
		app.NewMonitor(300)
	}

	log.Printf("%s version %s, commit %s\n", name, version, commit)

	log.Println("Starting instance")

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	loader.Start()
	for _, server := range servers {
		server.Start()
	}

	<-exit
	signal.Stop(exit)

	log.Println("Stopping instance")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer func() {
		cancel()
	}()

	for _, server := range servers {
		server.Stop(ctx)
	}

	return nil
}
