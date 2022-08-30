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
		rewrite, err := app.CreateRewriteRenderer(&configServer.Rewrite)
		if err != nil {
			log.Fatal(err)
		}

		static, err := app.CreateStaticRenderer(&configServer.Static)
		if err != nil {
			log.Fatal(err)
		}

		robots, err := app.CreateRobotsRenderer(&configServer.Robots, loader)
		if err != nil {
			log.Fatal(err)
		}

		sitemap, err := app.CreateSitemapRenderer(&configServer.Sitemap, fetcher)
		if err != nil {
			log.Fatal(err)
		}

		index, err := app.CreateIndexRenderer(&configServer.Index, fetcher)
		if err != nil {
			log.Fatal(err)
		}

		e, err := app.CreateErrorRenderer(&app.ErrorRendererConfig{StatusCode: configServer.ErrorCode})
		if err != nil {
			log.Fatal(err)
		}

		server, err := app.CreateServer(configServer, rewrite, static, robots, sitemap, index, e)
		if err != nil {
			log.Fatal(err)
		}

		servers = append(servers, server)
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Println("Starting instance")

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
