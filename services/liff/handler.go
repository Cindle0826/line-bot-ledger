package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
	"github.com/cindle0826/line-bot-ledger/internal/store"
)

// verifier checks a LIFF ID token and returns the LINE userId it belongs
// to. Narrowed to this one method so tests can inject a fake instead of
// hitting LINE's real verify endpoint.
type verifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (lineUserID string, err error)
}

// liffHandler handles requests from the LIFF ledger form and history page.
type liffHandler struct {
	verifier verifier
	store    store.Store
}

func newLiffHandler(v verifier, s store.Store) *liffHandler {
	return &liffHandler{verifier: v, store: s}
}

// authenticatedUser resolves the caller's LINE userId from its bearer
// token. Shared by every /liff/* endpoint — none of them trust Cloud Run's
// own IAM layer, they all verify the LIFF ID token themselves.
func (h *liffHandler) authenticatedUser(r *http.Request) (string, error) {
	idToken, ok := bearerToken(r)
	if !ok {
		return "", errors.New("liff: missing bearer token")
	}
	return h.verifier.VerifyIDToken(r.Context(), idToken)
}

type entryRequest struct {
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Note     string  `json:"note"`
}

// createEntry records one ledger entry submitted by the LIFF form. Unlike
// services/webhook's /callback, the caller here is the user's own browser,
// not LINE's platform — so identity comes from a LIFF ID token instead of
// an X-Line-Signature header.
func (h *liffHandler) createEntry(w http.ResponseWriter, r *http.Request) {
	lineUserID, err := h.authenticatedUser(r)
	if err != nil {
		slog.Warn("liff: authentication failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	entry, ok := decodeEntryRequest(w, r)
	if !ok {
		return
	}

	if err := h.store.AddEntry(r.Context(), lineUserID, entry); err != nil {
		slog.Error("liff: add entry failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// updateEntry corrects an existing entry's amount/category/note — same
// request shape as createEntry, but targets one entry by ID instead of
// creating a new one.
func (h *liffHandler) updateEntry(w http.ResponseWriter, r *http.Request) {
	lineUserID, err := h.authenticatedUser(r)
	if err != nil {
		slog.Warn("liff: authentication failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	entry, ok := decodeEntryRequest(w, r)
	if !ok {
		return
	}

	entryID := r.PathValue("id")
	if err := h.store.UpdateEntry(r.Context(), lineUserID, entryID, entry); err != nil {
		slog.Error("liff: update entry failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// deleteEntry removes one entry by ID.
func (h *liffHandler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	lineUserID, err := h.authenticatedUser(r)
	if err != nil {
		slog.Warn("liff: authentication failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	entryID := r.PathValue("id")
	if err := h.store.DeleteEntry(r.Context(), lineUserID, entryID); err != nil {
		slog.Error("liff: delete entry failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// decodeEntryRequest reads and validates the shared create/update request
// body, writing an error status itself and returning ok=false on failure.
func decodeEntryRequest(w http.ResponseWriter, r *http.Request) (ledger.Entry, bool) {
	var req entryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ledger.Entry{}, false
	}
	if req.Category == "" || req.Amount == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return ledger.Entry{}, false
	}
	return ledger.Entry{Amount: req.Amount, Category: req.Category, Note: req.Note}, true
}

type listEntriesResponse struct {
	Entries    []ledger.Entry         `json:"entries"`
	Categories []ledger.CategoryTotal `json:"categories"`
	Total      float64                `json:"total"`
}

// listEntries powers the history page: a date range (defaulting to today)
// plus optional category/sign filters, and the same category breakdown
// services/summary pushes daily — one query, one aggregation function,
// shared by both.
func (h *liffHandler) listEntries(w http.ResponseWriter, r *http.Request) {
	lineUserID, err := h.authenticatedUser(r)
	if err != nil {
		slog.Warn("liff: authentication failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	from, to, err := parseDateRange(q.Get("from"), q.Get("to"))
	if err != nil {
		slog.Warn("liff: bad date range", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	entries, err := h.store.ListEntries(r.Context(), lineUserID, from, to)
	if err != nil {
		slog.Error("liff: list entries failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	entries = filterEntries(entries, q.Get("category"), q.Get("sign"))
	if entries == nil {
		entries = []ledger.Entry{}
	}

	summary := ledger.Summarize(entries)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listEntriesResponse{
		Entries:    entries,
		Categories: summary.Categories,
		Total:      summary.Total,
	})
}

// parseDateRange reads "from"/"to" as Taipei-local calendar dates
// (YYYY-MM-DD), with "to" inclusive of that whole day. Both empty means
// "today". They're an all-or-nothing pair — a lone from/to is rejected
// rather than silently guessing the other end.
func parseDateRange(from, to string) (time.Time, time.Time, error) {
	if from == "" && to == "" {
		f, t := ledger.DayRange(time.Now())
		return f, t, nil
	}
	if from == "" || to == "" {
		return time.Time{}, time.Time{}, errors.New("liff: from and to must both be set")
	}

	fromDay, err := time.ParseInLocation("2006-01-02", from, ledger.Taipei)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("liff: invalid from date: %w", err)
	}
	toDay, err := time.ParseInLocation("2006-01-02", to, ledger.Taipei)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("liff: invalid to date: %w", err)
	}
	return fromDay, toDay.Add(24 * time.Hour), nil
}

// filterEntries applies the optional category ("" = any) and sign
// ("expense"/"income", "" = any) filters client-side — entry volume per
// user is small enough that a second Firestore query per filter isn't
// worth the complexity.
func filterEntries(entries []ledger.Entry, category, sign string) []ledger.Entry {
	if category == "" && sign == "" {
		return entries
	}
	filtered := make([]ledger.Entry, 0, len(entries))
	for _, e := range entries {
		if category != "" && e.Category != category {
			continue
		}
		if sign == "expense" && e.Amount >= 0 {
			continue
		}
		if sign == "income" && e.Amount <= 0 {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

type settingsResponse struct {
	Subscriptions []ledger.Frequency `json:"subscriptions"`
}

// getSettings returns the caller's current subscription frequencies, for
// the settings page to pre-check on load.
func (h *liffHandler) getSettings(w http.ResponseWriter, r *http.Request) {
	lineUserID, err := h.authenticatedUser(r)
	if err != nil {
		slog.Warn("liff: authentication failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	subs, err := h.store.GetSubscriptions(r.Context(), lineUserID)
	if err != nil {
		slog.Error("liff: get subscriptions failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if subs == nil {
		subs = []ledger.Frequency{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(settingsResponse{Subscriptions: subs})
}

type updateSettingsRequest struct {
	Subscriptions []ledger.Frequency `json:"subscriptions"`
}

// updateSettings replaces the caller's subscribed frequencies — a checkbox
// per frequency on the settings page, so this always sends the full set
// rather than adding/removing one at a time.
func (h *liffHandler) updateSettings(w http.ResponseWriter, r *http.Request) {
	lineUserID, err := h.authenticatedUser(r)
	if err != nil {
		slog.Warn("liff: authentication failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, f := range req.Subscriptions {
		if !f.Valid() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if err := h.store.SetSubscriptions(r.Context(), lineUserID, req.Subscriptions); err != nil {
		slog.Error("liff: set subscriptions failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	return strings.TrimPrefix(h, prefix), true
}
