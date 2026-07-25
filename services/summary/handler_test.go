package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
)

// fakeStore is an in-memory store.Store for tests: subs maps a lineUserID to
// its subscribed frequencies, entries maps a lineUserID to its ledger.
type fakeStore struct {
	subs    map[string][]ledger.Frequency
	entries map[string][]ledger.Entry
	listErr error
}

func (f *fakeStore) AddEntry(ctx context.Context, lineUserID string, entry ledger.Entry) error {
	return errors.New("not used in these tests")
}

func (f *fakeStore) ListEntries(ctx context.Context, lineUserID string, from, to time.Time) ([]ledger.Entry, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.entries[lineUserID], nil
}

func (f *fakeStore) UpdateEntry(ctx context.Context, lineUserID, entryID string, entry ledger.Entry) error {
	return errors.New("not used in these tests")
}

func (f *fakeStore) DeleteEntry(ctx context.Context, lineUserID, entryID string) error {
	return errors.New("not used in these tests")
}

func (f *fakeStore) ListSubscribers(ctx context.Context, frequency ledger.Frequency) ([]string, error) {
	var ids []string
	for uid, freqs := range f.subs {
		for _, fr := range freqs {
			if fr == frequency {
				ids = append(ids, uid)
				break
			}
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (f *fakeStore) GetSubscriptions(ctx context.Context, lineUserID string) ([]ledger.Frequency, error) {
	return f.subs[lineUserID], nil
}

func (f *fakeStore) SetSubscriptions(ctx context.Context, lineUserID string, frequencies []ledger.Frequency) error {
	if f.subs == nil {
		f.subs = map[string][]ledger.Frequency{}
	}
	f.subs[lineUserID] = frequencies
	return nil
}

func (f *fakeStore) Close() error { return nil }

type fakePusher struct {
	pushed []*messaging_api.PushMessageRequest
	errFor map[string]error
}

func (f *fakePusher) PushMessage(req *messaging_api.PushMessageRequest, retryKey string) (*messaging_api.PushMessageResponse, error) {
	if err := f.errFor[req.To]; err != nil {
		return nil, err
	}
	f.pushed = append(f.pushed, req)
	return &messaging_api.PushMessageResponse{}, nil
}

func (f *fakePusher) textFor(to string) string {
	for _, req := range f.pushed {
		if req.To == to {
			if msg, ok := req.Messages[0].(messaging_api.TextMessage); ok {
				return msg.Text
			}
		}
	}
	return ""
}

// aMonday/aTuesday/aFirstOfMonth are fixed instants so tests control which
// jobs run without depending on the real calendar.
var aMonday = time.Date(2026, time.January, 5, 21, 0, 0, 0, ledger.Taipei)
var aTuesday = time.Date(2026, time.January, 6, 21, 0, 0, 0, ledger.Taipei)
var aFirstOfMonth = time.Date(2026, time.February, 1, 21, 0, 0, 0, ledger.Taipei)

func TestRun_DailyPushesToSubscribersOnly(t *testing.T) {
	s := &fakeStore{
		subs: map[string][]ledger.Frequency{
			"U1": {ledger.FrequencyDaily},
			"U2": {ledger.FrequencyWeekly}, // not subscribed to daily
		},
		entries: map[string][]ledger.Entry{
			"U1": {{Amount: -100, Category: "午餐"}, {Amount: -50, Category: "交通"}},
		},
	}
	p := &fakePusher{}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aTuesday }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(p.pushed) != 1 {
		t.Fatalf("pushed %d messages, want 1 (only U1 is a daily subscriber)", len(p.pushed))
	}
	text := p.textFor("U1")
	for _, want := range []string{"午餐：-100", "交通：-50", "總計：-150"} {
		if !strings.Contains(text, want) {
			t.Errorf("push text %q missing %q", text, want)
		}
	}
}

func TestRun_WeeklyAndMonthlyOnlyRunOnTriggerDays(t *testing.T) {
	s := &fakeStore{subs: map[string][]ledger.Frequency{
		"U1": {ledger.FrequencyDaily, ledger.FrequencyWeekly, ledger.FrequencyMonthly},
	}}
	p := &fakePusher{}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aTuesday }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(p.pushed) != 1 {
		t.Fatalf("pushed %d messages on a non-trigger day, want 1 (daily only)", len(p.pushed))
	}
}

func TestRun_WeeklyRunsOnMonday(t *testing.T) {
	s := &fakeStore{subs: map[string][]ledger.Frequency{
		"U1": {ledger.FrequencyWeekly},
	}}
	p := &fakePusher{}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aMonday }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(p.pushed) != 1 {
		t.Fatalf("pushed %d messages on Monday, want 1 (weekly subscriber)", len(p.pushed))
	}
	if !strings.Contains(p.textFor("U1"), "本週") {
		t.Errorf("push text %q should mention 本週", p.textFor("U1"))
	}
}

func TestRun_MonthlyRunsOnFirstOfMonth(t *testing.T) {
	s := &fakeStore{subs: map[string][]ledger.Frequency{
		"U1": {ledger.FrequencyMonthly},
	}}
	p := &fakePusher{}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aFirstOfMonth }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(p.pushed) != 1 {
		t.Fatalf("pushed %d messages on the 1st, want 1 (monthly subscriber)", len(p.pushed))
	}
	if !strings.Contains(p.textFor("U1"), "2026年01月") {
		t.Errorf("push text %q should mention 2026年01月", p.textFor("U1"))
	}
}

func TestRun_NoSubscribersIsANoOp(t *testing.T) {
	s := &fakeStore{}
	p := &fakePusher{}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aTuesday }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(p.pushed) != 0 {
		t.Errorf("pushed %d messages, want 0", len(p.pushed))
	}
}

func TestRun_NoEntriesToday(t *testing.T) {
	s := &fakeStore{subs: map[string][]ledger.Frequency{"U1": {ledger.FrequencyDaily}}}
	p := &fakePusher{}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aTuesday }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(p.textFor("U1"), "還沒有記帳紀錄") {
		t.Errorf("push text %q should say there's nothing recorded yet", p.textFor("U1"))
	}
}

func TestRun_StoreErrorReturns500(t *testing.T) {
	s := &fakeStore{listErr: errors.New("boom"), subs: map[string][]ledger.Frequency{"U1": {ledger.FrequencyDaily}}}
	p := &fakePusher{}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aTuesday }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestRun_OnePushErrorDoesNotBlockOthers(t *testing.T) {
	s := &fakeStore{subs: map[string][]ledger.Frequency{
		"U1": {ledger.FrequencyDaily},
		"U2": {ledger.FrequencyDaily},
	}}
	p := &fakePusher{errFor: map[string]error{"U1": errors.New("blocked")}}
	h := &summaryHandler{store: s, pusher: p, now: func() time.Time { return aTuesday }}

	w := httptest.NewRecorder()
	h.run(w, httptest.NewRequest(http.MethodPost, "/run", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (one push failed)", w.Code)
	}
	if len(p.pushed) != 1 || p.pushed[0].To != "U2" {
		t.Errorf("expected U2 to still get pushed despite U1 failing, got %v", p.pushed)
	}
}
