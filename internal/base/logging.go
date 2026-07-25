package base

import (
	"log/slog"
	"os"
)

// init configures the default slog logger for every service that imports
// base (i.e. both of them), so main.go doesn't need to remember to set this
// up itself and logging is JSON — with a timestamp — from the very first
// line, including failures inside New before a service-specific logger
// could otherwise be wired up. Cloud Run/Cloud Logging parses JSON stdout
// lines directly, and the default handler already includes a "time" field.
func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}
