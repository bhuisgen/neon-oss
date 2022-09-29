// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/bhuisgen/neon/internal/app"
)

// checkCommand implements the check command
type checkCommand struct {
	flagset *flag.FlagSet
	timeout uint
	verbose bool
}

// NewCheckCommand creates the command
func NewCheckCommand() *checkCommand {
	c := checkCommand{}
	c.flagset = flag.NewFlagSet("check", flag.ExitOnError)
	c.flagset.UintVar(&c.timeout, "timeout", 5, "Set the check timeout (seconds)")
	c.flagset.BoolVar(&c.verbose, "verbose", false, "Use verbose output")
	c.flagset.Usage = func() {
		fmt.Println("Usage: neon check [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		c.flagset.PrintDefaults()
		os.Exit(2)
	}

	return &c
}

// Name returns the command name
func (c *checkCommand) Name() string {
	return c.flagset.Name()
}

// Description returns the command description
func (c *checkCommand) Description() string {
	return "Check the instance health"
}

// Init initializes the command
func (c *checkCommand) Init(args []string) error {
	return c.flagset.Parse(args)
}

// Execute executes the command
func (c *checkCommand) Execute() error {
	config, err := app.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %s\n", err)

		return err
	}

	for _, serverConfig := range config.Server {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if serverConfig.TLSCAFile != nil {
			ca, err := os.ReadFile(*serverConfig.TLSCAFile)
			if err != nil {
				fmt.Printf("Failed to read TLS CA file: %s\n", err)

				return err
			}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(ca)

			tlsConfig.RootCAs = caCertPool

			if serverConfig.TLSCertFile != nil && serverConfig.TLSKeyFile != nil {
				clientCert, err := tls.LoadX509KeyPair(*serverConfig.TLSCertFile, *serverConfig.TLSKeyFile)
				if err != nil {
					fmt.Printf("Failed to parse TLS certificate: %s\n", err)

					return err
				}

				tlsConfig.Certificates = []tls.Certificate{clientCert}
			}
		}

		transport := http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout: time.Duration(c.timeout) * time.Second,
			}).Dial,
			TLSClientConfig:       tlsConfig,
			TLSHandshakeTimeout:   time.Duration(c.timeout) * time.Second,
			ResponseHeaderTimeout: time.Duration(c.timeout) * time.Second,
			ExpectContinueTimeout: time.Duration(c.timeout) * time.Second,
			ForceAttemptHTTP2:     true,
		}

		client := http.Client{
			Transport: &transport,
			Timeout:   time.Duration(c.timeout) * time.Second,
		}

		scheme := "http"
		if serverConfig.TLS {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s:%d", scheme, serverConfig.ListenAddr, serverConfig.ListenPort)

		_, err := client.Head(url)
		if c.verbose {
			if err != nil {
				fmt.Printf("Check '%s': KO\n", url)
			} else {
				fmt.Printf("Check '%s': OK\n", url)
			}
		}
		if err != nil {
			return err
		}
	}

	return nil
}
