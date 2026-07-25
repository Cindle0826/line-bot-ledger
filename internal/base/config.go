// Package base bundles the environment configuration and initialized
// clients shared by every service in this repo. Each service calls New to
// get one, then wires its own service-specific pieces on top of it (e.g.
// webhook's channel secret + HTTP handler, summary's target user ID).
package base

import (
	"context"
	"fmt"
	"os"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"

	"github.com/cindle0826/line-bot-ledger/internal/store"
)

// Config holds the environment values every service needs, regardless of
// which Cloud Run service it's deployed as.
type Config struct {
	Port             string
	LineChannelToken string
	GCPProjectID     string
}

// BaseConfig bundles the common Config with the clients built from it, so a
// service gets everything it needs to talk to Firestore and LINE from one
// New call.
type BaseConfig struct {
	Cfg   Config
	Store store.Store
	Bot   *messaging_api.MessagingApiAPI
}

// New loads Config from the environment and initializes the Firestore and
// LINE clients built from it. The named return lets the deferred cleanup
// below tell success from failure: if a later step fails after fsStore was
// already opened, we close it here to avoid leaking the connection; on
// success it stays open, since the caller now owns it and closes it via
// BaseConfig.Close.
func New(ctx context.Context) (bc *BaseConfig, err error) {
	cfg := Config{Port: portOrDefault()}

	if cfg.LineChannelToken, err = RequireEnv("LINE_CHANNEL_TOKEN"); err != nil {
		return nil, err
	}
	if cfg.GCPProjectID, err = RequireEnv("GOOGLE_CLOUD_PROJECT"); err != nil {
		return nil, err
	}

	fsStore, err := store.NewFirestoreStore(ctx, cfg.GCPProjectID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = fsStore.Close()
		}
	}()

	bot, err := messaging_api.NewMessagingApiAPI(cfg.LineChannelToken)
	if err != nil {
		return nil, err
	}

	return &BaseConfig{Cfg: cfg, Store: fsStore, Bot: bot}, nil
}

// Close releases the resources held by BaseConfig.
func (b *BaseConfig) Close() error {
	return b.Store.Close()
}

// RequireEnv reads a required environment variable, exported so each
// service can load its own additional required vars the same way (e.g.
// webhook's LINE_CHANNEL_SECRET, liff's LIFF_ID).
func RequireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("config: %s is required", key)
	}
	return v, nil
}

func portOrDefault() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
