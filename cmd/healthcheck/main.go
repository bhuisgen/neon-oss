package main

import (
	"context"
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
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// run parses and executes the command line.
func run() error {
	var cacert, cert, key string
	var timeout, status int
	var verbose bool
	flag.StringVar(&cacert, "cacert", "", "TLS CA file")
	flag.StringVar(&cert, "cert", "", "TLS certificate file")
	flag.StringVar(&key, "key", "", "TLS key file")
	flag.IntVar(&status, "status", 0, "Status code")
	flag.IntVar(&timeout, "timeout", 5, "Timeout in seconds")
	flag.BoolVar(&verbose, "verbose", false, "Use verbose output")
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

	if err := healthcheck(flag.Arg(0), cacert, cert, key, status, timeout); err != nil {
		if verbose {
			fmt.Println("Error: ", err)
		}
		return fmt.Errorf("healthcheck: %v", err)
	}

	return nil
}

// healthcheck performs a request to the server endpoint.
func healthcheck(url string, cacert string, cert string, key string, status int, timeout int) error {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cacert != "" {
		ca, err := os.ReadFile(cacert)
		if err != nil {
			return fmt.Errorf("read ca file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(ca) {
			return errors.New("append ca")
		}

		tlsConfig.RootCAs = caCertPool

		if cert != "" && key != "" {
			c, err := tls.LoadX509KeyPair(cert, key)
			if err != nil {
				return fmt.Errorf("load keypair: %v", err)
			}

			tlsConfig.Certificates = []tls.Certificate{c}
		}
	}
	client := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout: time.Duration(timeout) * time.Second,
			}).Dial,
			TLSClientConfig:       tlsConfig,
			TLSHandshakeTimeout:   time.Duration(timeout) * time.Second,
			ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
			ExpectContinueTimeout: time.Duration(timeout) * time.Second,
			ForceAttemptHTTP2:     true,
		},
		Timeout: time.Duration(timeout) * time.Second,
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %v", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("read response: %v", err)
	}

	if status > 0 {
		if response.StatusCode != status {
			return fmt.Errorf("status code: %d", response.StatusCode)
		}
	}

	return nil
}
