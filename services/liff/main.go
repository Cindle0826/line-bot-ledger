// Command liff serves the LIFF ledger-entry form, the history page, and
// the API both submit to. It is deployed as its own Cloud Run service,
// separate from services/webhook and services/summary: the caller here is
// the user's own browser (via LINE's in-app browser), not LINE's Messaging
// API platform, so it has a different trigger and a different auth
// mechanism (LIFF ID token instead of a webhook signature).
package main

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"os"

	"github.com/cindle0826/line-bot-ledger/internal/base"
)

//go:embed static/index.html.tmpl static/history.html.tmpl static/settings.html.tmpl
var staticFiles embed.FS

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

	channelID, err := base.RequireEnv("LINE_LOGIN_CHANNEL_ID")
	if err != nil {
		slog.Error("missing required env", "error", err)
		os.Exit(1)
	}
	liffID, err := base.RequireEnv("LIFF_ID")
	if err != nil {
		slog.Error("missing required env", "error", err)
		os.Exit(1)
	}

	indexPage, err := renderPage("static/index.html.tmpl", liffID)
	if err != nil {
		slog.Error("render index page failed", "error", err)
		os.Exit(1)
	}
	historyPage, err := renderPage("static/history.html.tmpl", liffID)
	if err != nil {
		slog.Error("render history page failed", "error", err)
		os.Exit(1)
	}
	settingsPage, err := renderPage("static/settings.html.tmpl", liffID)
	if err != nil {
		slog.Error("render settings page failed", "error", err)
		os.Exit(1)
	}

	h := newLiffHandler(newLineVerifier(channelID), b.Store)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", servePage(indexPage))
	mux.HandleFunc("GET /history", servePage(historyPage))
	mux.HandleFunc("GET /settings", servePage(settingsPage))
	mux.HandleFunc("POST /liff/entries", h.createEntry)
	mux.HandleFunc("GET /liff/entries", h.listEntries)
	mux.HandleFunc("PUT /liff/entries/{id}", h.updateEntry)
	mux.HandleFunc("DELETE /liff/entries/{id}", h.deleteEntry)
	mux.HandleFunc("GET /liff/settings", h.getSettings)
	mux.HandleFunc("POST /liff/settings", h.updateSettings)
	mux.HandleFunc("/status", base.Healthz)

	slog.Info("listening", "port", b.Cfg.Port)
	if err := http.ListenAndServe(":"+b.Cfg.Port, mux); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func servePage(page []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	}
}

// renderPage renders a LIFF page template once at startup — the LIFF ID
// never changes at runtime, so there's no reason to re-render it per
// request.
func renderPage(name, liffID string) ([]byte, error) {
	tmpl, err := template.ParseFS(staticFiles, name)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ LiffID string }{LiffID: liffID}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
