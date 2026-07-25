// Package store persists ledger entries.
package store

import (
	"context"
	"time"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
)

// Store saves and queries bookkeeping entries for a LINE user.
type Store interface {
	AddEntry(ctx context.Context, lineUserID string, entry ledger.Entry) error
	// ListEntries returns entries created in [from, to), oldest first, each
	// with its ID populated.
	ListEntries(ctx context.Context, lineUserID string, from, to time.Time) ([]ledger.Entry, error)
	// UpdateEntry overwrites entryID's amount/category/note, leaving
	// createdAt untouched — it's a correction, not a re-dated entry.
	UpdateEntry(ctx context.Context, lineUserID, entryID string, entry ledger.Entry) error
	// DeleteEntry removes entryID.
	DeleteEntry(ctx context.Context, lineUserID, entryID string) error
	// ListSubscribers returns the LINE userIds subscribed to frequency —
	// services/summary's daily run fans out over these per job.
	ListSubscribers(ctx context.Context, frequency ledger.Frequency) ([]string, error)
	// GetSubscriptions returns lineUserID's subscribed frequencies (nil if
	// they haven't subscribed to anything).
	GetSubscriptions(ctx context.Context, lineUserID string) ([]ledger.Frequency, error)
	// SetSubscriptions replaces lineUserID's subscribed frequencies.
	SetSubscriptions(ctx context.Context, lineUserID string, frequencies []ledger.Frequency) error
	Close() error
}
