// Command summary runs the daily ledger summary service on Cloud Run.
// Unlike services/webhook, it has no public webhook contract: Cloud
// Scheduler calls /run once a day, and it fans out over every user
// subscribed (daily/weekly/monthly, set via services/liff's settings page)
// to that day's frequencies, pushing each their own summary via the LINE
// Messaging API push endpoint (not a reply, since there's no webhook
// event/reply token here).
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

	h := newSummaryHandler(b.Store, b.Bot)
	mux := http.NewServeMux()
	mux.HandleFunc("/run", h.run)
	mux.HandleFunc("/status", base.Healthz)

	slog.Info("listening", "port", b.Cfg.Port)
	if err := http.ListenAndServe(":"+b.Cfg.Port, mux); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
