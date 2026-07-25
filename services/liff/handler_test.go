package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
)

type fakeVerifier struct {
	userID string
	err    error
}

func (f fakeVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	return f.userID, f.err
}

type fakeStore struct {
	entries    map[string][]ledger.Entry
	addErr     error
	listErr    error
	updateErr  error
	deleteErr  error
	subs       map[string][]ledger.Frequency
	getSubsErr error
	setSubsErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{entries: map[string][]ledger.Entry{}, subs: map[string][]ledger.Frequency{}}
}

func (f *fakeStore) AddEntry(ctx context.Context, lineUserID string, entry ledger.Entry) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.entries[lineUserID] = append(f.entries[lineUserID], entry)
	return nil
}

func (f *fakeStore) ListEntries(ctx context.Context, lineUserID string, from, to time.Time) ([]ledger.Entry, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []ledger.Entry
	for _, e := range f.entries[lineUserID] {
		if !e.CreatedAt.Before(from) && e.CreatedAt.Before(to) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeStore) UpdateEntry(ctx context.Context, lineUserID, entryID string, entry ledger.Entry) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	for i, e := range f.entries[lineUserID] {
		if e.ID == entryID {
			entry.ID = entryID
			entry.CreatedAt = e.CreatedAt
			f.entries[lineUserID][i] = entry
			return nil
		}
	}
	return errors.New("entry not found")
}

func (f *fakeStore) DeleteEntry(ctx context.Context, lineUserID, entryID string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	out := f.entries[lineUserID][:0]
	for _, e := range f.entries[lineUserID] {
		if e.ID != entryID {
			out = append(out, e)
		}
	}
	f.entries[lineUserID] = out
	return nil
}

func (f *fakeStore) ListSubscribers(ctx context.Context, frequency ledger.Frequency) ([]string, error) {
	return nil, nil
}

func (f *fakeStore) GetSubscriptions(ctx context.Context, lineUserID string) ([]ledger.Frequency, error) {
	if f.getSubsErr != nil {
		return nil, f.getSubsErr
	}
	return f.subs[lineUserID], nil
}

func (f *fakeStore) SetSubscriptions(ctx context.Context, lineUserID string, frequencies []ledger.Frequency) error {
	if f.setSubsErr != nil {
		return f.setSubsErr
	}
	f.subs[lineUserID] = frequencies
	return nil
}

func (f *fakeStore) Close() error { return nil }

func TestCreateEntry(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		verifier   fakeVerifier
		body       string
		storeErr   error
		wantStatus int
		wantSaved  bool
	}{
		{
			name:       "valid expense",
			authHeader: "Bearer good-token",
			verifier:   fakeVerifier{userID: "U123"},
			body:       `{"amount":-100,"category":"午餐","note":"便當"}`,
			wantStatus: http.StatusOK,
			wantSaved:  true,
		},
		{
			name:       "valid income",
			authHeader: "Bearer good-token",
			verifier:   fakeVerifier{userID: "U123"},
			body:       `{"amount":5000,"category":"薪水"}`,
			wantStatus: http.StatusOK,
			wantSaved:  true,
		},
		{
			name:       "missing auth header",
			authHeader: "",
			verifier:   fakeVerifier{userID: "U123"},
			body:       `{"amount":-100,"category":"午餐"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer bad-token",
			verifier:   fakeVerifier{err: errors.New("boom")},
			body:       `{"amount":-100,"category":"午餐"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing category",
			authHeader: "Bearer good-token",
			verifier:   fakeVerifier{userID: "U123"},
			body:       `{"amount":-100}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero amount",
			authHeader: "Bearer good-token",
			verifier:   fakeVerifier{userID: "U123"},
			body:       `{"amount":0,"category":"午餐"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "store failure",
			authHeader: "Bearer good-token",
			verifier:   fakeVerifier{userID: "U123"},
			body:       `{"amount":-100,"category":"午餐"}`,
			storeErr:   errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newFakeStore()
			s.addErr = tt.storeErr
			h := newLiffHandler(tt.verifier, s)

			req := httptest.NewRequest(http.MethodPost, "/liff/entries", bytes.NewBufferString(tt.body))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			h.createEntry(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantSaved && len(s.entries["U123"]) != 1 {
				t.Errorf("expected 1 entry saved for U123, got %d", len(s.entries["U123"]))
			}
		})
	}
}

func TestUpdateEntry(t *testing.T) {
	s := newFakeStore()
	s.entries["U123"] = []ledger.Entry{
		{ID: "e1", Amount: -100, Category: "午餐", CreatedAt: time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)},
	}
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodPut, "/liff/entries/e1", bytes.NewBufferString(`{"amount":-120,"category":"午餐","note":"改過"}`))
	req.Header.Set("Authorization", "Bearer good-token")
	req.SetPathValue("id", "e1")
	rec := httptest.NewRecorder()
	h.updateEntry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	got := s.entries["U123"][0]
	if got.Amount != -120 || got.Note != "改過" {
		t.Errorf("entry = %+v, want amount=-120 note=改過", got)
	}
	if !got.CreatedAt.Equal(time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("CreatedAt = %v, should be untouched by an update", got.CreatedAt)
	}
}

func TestUpdateEntry_InvalidBody(t *testing.T) {
	s := newFakeStore()
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodPut, "/liff/entries/e1", bytes.NewBufferString(`{"amount":0,"category":"午餐"}`))
	req.Header.Set("Authorization", "Bearer good-token")
	req.SetPathValue("id", "e1")
	rec := httptest.NewRecorder()
	h.updateEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateEntry_Unauthorized(t *testing.T) {
	h := newLiffHandler(fakeVerifier{err: errors.New("boom")}, newFakeStore())

	req := httptest.NewRequest(http.MethodPut, "/liff/entries/e1", bytes.NewBufferString(`{"amount":-100,"category":"午餐"}`))
	req.Header.Set("Authorization", "Bearer bad-token")
	req.SetPathValue("id", "e1")
	rec := httptest.NewRecorder()
	h.updateEntry(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestDeleteEntry(t *testing.T) {
	s := newFakeStore()
	s.entries["U123"] = []ledger.Entry{
		{ID: "e1", Amount: -100, Category: "午餐"},
		{ID: "e2", Amount: -50, Category: "交通"},
	}
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodDelete, "/liff/entries/e1", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	req.SetPathValue("id", "e1")
	rec := httptest.NewRecorder()
	h.deleteEntry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if len(s.entries["U123"]) != 1 || s.entries["U123"][0].ID != "e2" {
		t.Errorf("entries = %+v, want just e2 left", s.entries["U123"])
	}
}

func TestDeleteEntry_StoreError(t *testing.T) {
	s := newFakeStore()
	s.deleteErr = errors.New("boom")
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodDelete, "/liff/entries/e1", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	req.SetPathValue("id", "e1")
	rec := httptest.NewRecorder()
	h.deleteEntry(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestDeleteEntry_Unauthorized(t *testing.T) {
	h := newLiffHandler(fakeVerifier{err: errors.New("boom")}, newFakeStore())

	req := httptest.NewRequest(http.MethodDelete, "/liff/entries/e1", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	req.SetPathValue("id", "e1")
	rec := httptest.NewRecorder()
	h.deleteEntry(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestListEntries(t *testing.T) {
	s := newFakeStore()
	s.entries["U123"] = []ledger.Entry{
		{Amount: -100, Category: "午餐", CreatedAt: time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)},
		{Amount: -50, Category: "交通", CreatedAt: time.Date(2026, 7, 25, 4, 0, 0, 0, time.UTC)},
		{Amount: 5000, Category: "薪水", CreatedAt: time.Date(2026, 7, 25, 5, 0, 0, 0, time.UTC)},
	}
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodGet, "/liff/entries?from=2026-07-25&to=2026-07-25", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	h.listEntries(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got listEntriesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(got.Entries))
	}
	if got.Total != 4850 {
		t.Errorf("Total = %v, want 4850", got.Total)
	}
}

func TestListEntries_CategoryAndSignFilter(t *testing.T) {
	s := newFakeStore()
	s.entries["U123"] = []ledger.Entry{
		{Amount: -100, Category: "午餐", CreatedAt: time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)},
		{Amount: -50, Category: "交通", CreatedAt: time.Date(2026, 7, 25, 4, 0, 0, 0, time.UTC)},
		{Amount: 5000, Category: "午餐", CreatedAt: time.Date(2026, 7, 25, 5, 0, 0, 0, time.UTC)},
	}
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodGet, "/liff/entries?from=2026-07-25&to=2026-07-25&category=午餐&sign=expense", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	h.listEntries(rec, req)

	var got listEntriesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Amount != -100 {
		t.Errorf("Entries = %+v, want just the -100 午餐 expense", got.Entries)
	}
}

func TestListEntries_Unauthorized(t *testing.T) {
	h := newLiffHandler(fakeVerifier{err: errors.New("boom")}, newFakeStore())

	req := httptest.NewRequest(http.MethodGet, "/liff/entries", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	h.listEntries(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestParseDateRange(t *testing.T) {
	t.Run("defaults to today when both empty", func(t *testing.T) {
		from, to, err := parseDateRange("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !to.After(from) {
			t.Errorf("to (%v) should be after from (%v)", to, from)
		}
	})

	t.Run("inclusive end date", func(t *testing.T) {
		from, to, err := parseDateRange("2026-07-01", "2026-07-03")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantFrom := time.Date(2026, 7, 1, 0, 0, 0, 0, ledger.Taipei)
		wantTo := time.Date(2026, 7, 4, 0, 0, 0, 0, ledger.Taipei)
		if !from.Equal(wantFrom) || !to.Equal(wantTo) {
			t.Errorf("got (%v, %v), want (%v, %v)", from, to, wantFrom, wantTo)
		}
	})

	t.Run("rejects a lone from or to", func(t *testing.T) {
		if _, _, err := parseDateRange("2026-07-01", ""); err == nil {
			t.Error("expected an error for a lone 'from'")
		}
		if _, _, err := parseDateRange("", "2026-07-01"); err == nil {
			t.Error("expected an error for a lone 'to'")
		}
	})

	t.Run("rejects malformed dates", func(t *testing.T) {
		if _, _, err := parseDateRange("not-a-date", "2026-07-01"); err == nil {
			t.Error("expected an error for a malformed 'from'")
		}
	})
}

func TestFilterEntries(t *testing.T) {
	entries := []ledger.Entry{
		{Amount: -100, Category: "午餐"},
		{Amount: -50, Category: "交通"},
		{Amount: 5000, Category: "薪水"},
	}

	if got := filterEntries(entries, "", ""); len(got) != 3 {
		t.Errorf("no filters: len = %d, want 3", len(got))
	}
	if got := filterEntries(entries, "午餐", ""); len(got) != 1 {
		t.Errorf("category filter: len = %d, want 1", len(got))
	}
	if got := filterEntries(entries, "", "income"); len(got) != 1 || got[0].Category != "薪水" {
		t.Errorf("income filter: got %+v", got)
	}
	if got := filterEntries(entries, "", "expense"); len(got) != 2 {
		t.Errorf("expense filter: len = %d, want 2", len(got))
	}
}

func TestGetSettings(t *testing.T) {
	s := newFakeStore()
	s.subs["U123"] = []ledger.Frequency{ledger.FrequencyDaily, ledger.FrequencyMonthly}
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodGet, "/liff/settings", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	h.getSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got settingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Subscriptions) != 2 {
		t.Errorf("Subscriptions = %+v, want 2 entries", got.Subscriptions)
	}
}

func TestGetSettings_Unauthorized(t *testing.T) {
	h := newLiffHandler(fakeVerifier{err: errors.New("boom")}, newFakeStore())

	req := httptest.NewRequest(http.MethodGet, "/liff/settings", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	h.getSettings(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestUpdateSettings(t *testing.T) {
	s := newFakeStore()
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodPost, "/liff/settings", bytes.NewBufferString(`{"subscriptions":["daily","weekly"]}`))
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	h.updateSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if got := s.subs["U123"]; len(got) != 2 {
		t.Errorf("subs[U123] = %+v, want 2 entries", got)
	}
}

func TestUpdateSettings_RejectsInvalidFrequency(t *testing.T) {
	s := newFakeStore()
	h := newLiffHandler(fakeVerifier{userID: "U123"}, s)

	req := httptest.NewRequest(http.MethodPost, "/liff/settings", bytes.NewBufferString(`{"subscriptions":["yearly"]}`))
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	h.updateSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateSettings_Unauthorized(t *testing.T) {
	h := newLiffHandler(fakeVerifier{err: errors.New("boom")}, newFakeStore())

	req := httptest.NewRequest(http.MethodPost, "/liff/settings", bytes.NewBufferString(`{"subscriptions":["daily"]}`))
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	h.updateSettings(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestBearerToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/liff/entries", nil)
	if _, ok := bearerToken(req); ok {
		t.Error("expected no token when Authorization header is absent")
	}

	req.Header.Set("Authorization", "Bearer abc.def.ghi")
	token, ok := bearerToken(req)
	if !ok || token != "abc.def.ghi" {
		t.Errorf("got (%q, %v), want (%q, true)", token, ok, "abc.def.ghi")
	}

	req.Header.Set("Authorization", "abc.def.ghi")
	if _, ok := bearerToken(req); ok {
		t.Error("expected no token when Authorization header lacks Bearer prefix")
	}
}
