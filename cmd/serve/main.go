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

// main is the entrypoint
func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "show version information")

	flag.Parse()

	if showVersion {
		fmt.Printf("%s version %s, commit %s\n", name, version, commit)

		os.Exit(0)
	}

	config, err := app.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	var servers []*app.Server

	fetcher := app.NewFetcher(config.Fetcher)
	loader := app.NewLoader(config.Loader, fetcher)

	for _, configServer := range config.Server {
		var renderers []app.Renderer

		if configServer.Rewrite.Enable {
			rewrite, err := app.CreateRewriteRenderer(&configServer.Rewrite)
			if err != nil {
				log.Fatal(err)
			}
			renderers = append(renderers, rewrite)
		}

		if configServer.Static.Enable {
			header, err := app.CreateHeaderRenderer(&configServer.Header)
			if err != nil {
				log.Fatal(err)
			}
			renderers = append(renderers, header)
		}

		if configServer.Static.Enable {
			static, err := app.CreateStaticRenderer(&configServer.Static)
			if err != nil {
				log.Fatal(err)
			}
			renderers = append(renderers, static)
		}

		if configServer.Robots.Enable {
			robots, err := app.CreateRobotsRenderer(&configServer.Robots, loader)
			if err != nil {
				log.Fatal(err)
			}
			renderers = append(renderers, robots)
		}

		if configServer.Sitemap.Enable {
			sitemap, err := app.CreateSitemapRenderer(&configServer.Sitemap, fetcher)
			if err != nil {
				log.Fatal(err)
			}
			renderers = append(renderers, sitemap)
		}

		if configServer.Index.Enable {
			index, err := app.CreateIndexRenderer(&configServer.Index, fetcher)
			if err != nil {
				log.Fatal(err)
			}
			renderers = append(renderers, index)
		}

		e, err := app.CreateErrorRenderer(&app.ErrorRendererConfig{StatusCode: configServer.ErrorCode})
		if err != nil {
			log.Fatal(err)
		}
		renderers = append(renderers, e)

		server, err := app.CreateServer(configServer, renderers...)
		if err != nil {
			log.Fatal(err)
		}

		servers = append(servers, server)
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Println("Starting instance")

	if _, ok := os.LookupEnv("DEBUG"); ok {
		app.NewMonitor(300)
	}

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
}
