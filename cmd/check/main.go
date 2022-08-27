// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/bhuisgen/neon/internal/app"
)

// main is the entrypoint
func main() {
	config, err := app.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	for _, serverConfig := range config.Server {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if serverConfig.TLSCAFile != "" {
			ca, err := os.ReadFile(serverConfig.TLSCAFile)
			if err != nil {
				log.Fatal(err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(ca) {
				log.Fatal(err)
			}

			tlsConfig.RootCAs = caCertPool

			if serverConfig.TLSCertFile != "" && serverConfig.TLSKeyFile != "" {
				clientCert, err := tls.LoadX509KeyPair(serverConfig.TLSCertFile, serverConfig.TLSKeyFile)
				if err != nil {
					log.Fatal(err)
				}

				tlsConfig.Certificates = []tls.Certificate{clientCert}
			}
		}

		transport := http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSClientConfig:       tlsConfig,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 5 * time.Second,
			ForceAttemptHTTP2:     true,
		}

		client := http.Client{
			Transport: &transport,
			Timeout:   5 * time.Second,
		}

		scheme := "http"
		if serverConfig.TLS {
			scheme = "https"
		}
		addr := fmt.Sprintf("%s:%d", serverConfig.ListenAddr, serverConfig.ListenPort)

		_, err = client.Head(scheme + "://" + addr + "/status")
		if err != nil {
			log.Fatal(err)
		}
	}
}
