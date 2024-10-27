package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/alex-kennedy/up-bank-exporter/up"
)

var (
	port                 = flag.Int("port", 3000, "Port to serve /metrics HTTP handler.")
	upBearerTokenPath    = flag.String("up_bank_bearer_token_path", "/up/token.key", "Path to the Up API bearer token to use with the API. See https://developer.up.com.au/#authentication.")
	webhookSecretKeyPath = flag.String("up_bank_webhook_secret_key_path", "", "Path to an Up webhook secret key for authenticating received webhook requests. See https://developer.up.com.au/#callback_post_webhookURL.")
)

func main() {
	flag.Parse()

	if *upBearerTokenPath == "" {
		log.Fatalf("--up_bank_bearer_token_path is required but not set")
	}
	upBearerToken, err := os.ReadFile(*upBearerTokenPath)
	if err != nil {
		log.Fatalf("failed to read up bearer token path at %s: %v", *upBearerTokenPath, err)
	}

	handler, err := up.NewMetricsHandler(strings.TrimSpace(string(upBearerToken)))
	if err != nil {
		log.Fatalf("NewMetricsHandler() failed: %v", err)
	}
	http.Handle("/metrics", handler)

	if *webhookSecretKeyPath != "" {
		if err := registerWebhookHandler(); err != nil {
			log.Fatalf("failed to start webhook server: %v", err)
		}
	}

	log.Printf("starting metrics server on port %d\n", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if errors.Is(err, http.ErrServerClosed) {
		log.Println("shutting down server")
	} else {
		log.Fatalln(err)
	}
}

func registerWebhookHandler() error {
	key, err := os.ReadFile(*webhookSecretKeyPath)
	if err != nil {
		log.Fatalf("failed to read webhook secret key: %v", err)
	}
	key = bytes.TrimSpace(key)
	handler := up.NewUpWebhookHandler(key)
	http.Handle("/webhook", handler)
	return nil
}
