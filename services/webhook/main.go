// Command webhook receives LINE Messaging API events and records ledger
// entries. It is deployed as its own Cloud Run service, separate from
// services/summary.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/cindle0826/line-bot-ledger/internal/base"
)

func main() {
	ctx := context.Background()
	b, err := base.New(ctx)
	if err != nil {
		slog.Error("base.New failed", "error", err)
		os.Exit(1)
	}
	defer func(b *base.BaseConfig) {
		_ = b.Close()
	}(b)

	channelSecret, err := base.RequireEnv("LINE_CHANNEL_SECRET")
	if err != nil {
		slog.Error("missing required env", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/callback", newWebhookHandler(channelSecret, b.Bot, b.Store))
	mux.HandleFunc("/healthz", base.Healthz)

	slog.Info("listening", "port", b.Cfg.Port)
	if err := http.ListenAndServe(":"+b.Cfg.Port, mux); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
