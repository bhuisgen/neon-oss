// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// main is the entrypoint.
func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

// run parses and executes the command line.
func run() error {
	var cacert, cert, key string
	var timeout, status int
	flag.StringVar(&cacert, "cacert", "", "TLS CA file")
	flag.StringVar(&cert, "cert", "", "TLS certificate file")
	flag.StringVar(&key, "key", "", "TLS key file")
	flag.IntVar(&status, "status", 0, "Status code")
	flag.IntVar(&timeout, "timeout", 5, "Timeout in seconds")
	flag.Usage = func() {
		fmt.Println()
		fmt.Println("Usage: healthcheck [OPTIONS] [url]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Run 'healthcheck --help' for more information.")
		fmt.Println()
	}
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		return nil
	}

	err := healthcheck(flag.Arg(0), cacert, cert, key, status, timeout)
	if err != nil {
		return err
	}

	return nil
}

// healthcheck performs a request to the server endpoint.
func healthcheck(url string, cacert string, cert string, key string, status int, timeout int) error {
	tlsConfig := &tls.Config{}

	if cacert != "" {
		ca, err := os.ReadFile(cacert)
		if err != nil {
			fmt.Printf("Failed to read TLS CA file: %s\n", err)

			return err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(ca)

		tlsConfig.RootCAs = caCertPool

		if cert != "" && key != "" {
			c, err := tls.LoadX509KeyPair(cert, key)
			if err != nil {
				fmt.Printf("Failed to parse TLS certificate: %s\n", err)

				return err
			}

			tlsConfig.Certificates = []tls.Certificate{c}
		}
	}

	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: time.Duration(timeout) * time.Second,
		}).Dial,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   time.Duration(timeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
		ExpectContinueTimeout: time.Duration(timeout) * time.Second,
		ForceAttemptHTTP2:     true,
	}

	client := http.Client{
		Transport: &transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	response, err := client.Head(url)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return err
	}
	io.Copy(io.Discard, response.Body)
	if status > 0 {
		if response.StatusCode != status {
			return errors.New("invalid status")
		}
	}

	return nil
}
