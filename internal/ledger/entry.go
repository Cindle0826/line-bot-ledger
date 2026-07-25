// Package ledger implements parsing and storage of bookkeeping entries.
package ledger

import "time"

// Entry is one bookkeeping record, keyed by the LINE user who reported it.
// JSON tags exist alongside the Firestore ones so services/liff can hand
// entries straight back to the browser without a separate DTO.
type Entry struct {
	// ID is the Firestore document ID — not a stored field (it's the doc's
	// own identity), populated by the store on read so callers can target
	// UpdateEntry/DeleteEntry at a specific entry.
	ID        string    `firestore:"-" json:"id"`
	Amount    float64   `firestore:"amount" json:"amount"`
	Category  string    `firestore:"category" json:"category"`
	Note      string    `firestore:"note" json:"note"`
	CreatedAt time.Time `firestore:"createdAt,serverTimestamp" json:"createdAt"`
}
