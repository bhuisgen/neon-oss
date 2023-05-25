// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
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

// NewServeCommand creates a command
func NewServeCommand() *serveCommand {
	c := serveCommand{}
	c.flagset = flag.NewFlagSet("serve", flag.ExitOnError)
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon serve [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
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
func (c *serveCommand) Execute() error {
	config, err := app.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %s\n", err)

		return err
	}

	_, err = app.CheckConfig(config)
	if err != nil {
		fmt.Printf("Failed to validate configuration: %s\n", err)

		return err
	}

	var servers []app.Server

	fetcher, err := app.CreateFetcher(config.Fetcher)
	if err != nil {
		fmt.Printf("Failed to create fetcher: %s\n", err)
		return err
	}

	var loader app.Loader
	if config.Loader != nil {
		loader, err = app.CreateLoader(config.Loader, fetcher)
		if err != nil {
			fmt.Printf("Failed to create loader: %s\n", err)
			return err
		}
	}

	for _, configServer := range config.Server {
		var renderers []app.Renderer

		if configServer.Renderer.Rewrite != nil {
			rewrite, err := app.CreateRewriteRenderer(configServer.Renderer.Rewrite)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, rewrite)
		}

		if configServer.Renderer.Header != nil {
			header, err := app.CreateHeaderRenderer(configServer.Renderer.Header)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}

			renderers = append(renderers, header)
		}

		if configServer.Renderer.Static != nil {
			static, err := app.CreateStaticRenderer(configServer.Renderer.Static)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, static)
		}

		if configServer.Renderer.Robots != nil {
			robots, err := app.CreateRobotsRenderer(configServer.Renderer.Robots)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, robots)
		}

		if configServer.Renderer.Sitemap != nil {
			sitemap, err := app.CreateSitemapRenderer(configServer.Renderer.Sitemap, fetcher)
			if err != nil {
				fmt.Printf("Failed to create server: %s\n", err)
				return err
			}
			renderers = append(renderers, sitemap)
		}

		if configServer.Renderer.Index != nil {
			index, err := app.CreateIndexRenderer(configServer.Renderer.Index, fetcher)
			if err != nil {
				fmt.Printf("Failed to create index renderer: %s\n", err)
				return err
			}
			renderers = append(renderers, index)
		}

		if configServer.Renderer.Default != nil {
			d, err := app.CreateDefaultRenderer(configServer.Renderer.Default)
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

	if app.DEBUG {
		app.NewMonitor(&app.MonitorConfig{Delay: 300})
	}

	log.Printf("%s version %s, commit %s\n", app.Name, app.Version, app.Commit)

	log.Print("Starting instance")

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for _, server := range servers {
		server.Start()
	}

	if loader != nil {
		loader.Start()
	}

	<-exit
	signal.Stop(exit)

	log.Print("Stopping instance")

	if loader != nil {
		loader.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer func() {
		cancel()
	}()

	for _, server := range servers {
		server.Stop(ctx)
	}

	return nil
}
