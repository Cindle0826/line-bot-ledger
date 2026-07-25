package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
)

const testChannelSecret = "test-channel-secret"

// fakeStore is an in-memory store.Store for tests.
type fakeStore struct {
	entries map[string][]ledger.Entry
	addErr  error
}

func newFakeStore() *fakeStore {
	return &fakeStore{entries: map[string][]ledger.Entry{}}
}

func (s *fakeStore) AddEntry(_ context.Context, lineUserID string, entry ledger.Entry) error {
	if s.addErr != nil {
		return s.addErr
	}
	s.entries[lineUserID] = append(s.entries[lineUserID], entry)
	return nil
}

func (s *fakeStore) ListEntries(_ context.Context, lineUserID string, from, to time.Time) ([]ledger.Entry, error) {
	return s.entries[lineUserID], nil
}

func (s *fakeStore) UpdateEntry(_ context.Context, lineUserID, entryID string, entry ledger.Entry) error {
	return nil
}

func (s *fakeStore) DeleteEntry(_ context.Context, lineUserID, entryID string) error {
	return nil
}

func (s *fakeStore) ListSubscribers(_ context.Context, frequency ledger.Frequency) ([]string, error) {
	return nil, nil
}

func (s *fakeStore) GetSubscriptions(_ context.Context, lineUserID string) ([]ledger.Frequency, error) {
	return nil, nil
}

func (s *fakeStore) SetSubscriptions(_ context.Context, lineUserID string, frequencies []ledger.Frequency) error {
	return nil
}

func (s *fakeStore) Close() error { return nil }

// fakeReplier is an in-memory replier for tests, capturing every call
// instead of hitting the real LINE API.
type fakeReplier struct {
	calls    []*messaging_api.ReplyMessageRequest
	replyErr error
}

func (r *fakeReplier) ReplyMessage(req *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error) {
	r.calls = append(r.calls, req)
	if r.replyErr != nil {
		return nil, r.replyErr
	}
	return &messaging_api.ReplyMessageResponse{}, nil
}

func (r *fakeReplier) lastText() string {
	if len(r.calls) == 0 {
		return ""
	}
	msgs := r.calls[len(r.calls)-1].Messages
	if len(msgs) == 0 {
		return ""
	}
	tm, ok := msgs[0].(messaging_api.TextMessage)
	if !ok {
		return ""
	}
	return tm.Text
}

func TestRecordEntry(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		storeErr   error
		wantSubstr string
		wantSaved  bool
	}{
		{
			name:       "valid expense",
			text:       "-100 午餐",
			wantSubstr: "已記錄 支出：-100（午餐）",
			wantSaved:  true,
		},
		{
			name:       "valid income",
			text:       "+5000 薪水",
			wantSubstr: "已記錄 收入：5000（薪水）",
			wantSaved:  true,
		},
		{
			name:       "unparseable message",
			text:       "hello",
			wantSubstr: "看不懂這筆記帳",
			wantSaved:  false,
		},
		{
			name:       "store failure",
			text:       "-100 午餐",
			storeErr:   errors.New("firestore unavailable"),
			wantSubstr: "記帳失敗",
			wantSaved:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newFakeStore()
			s.addErr = tc.storeErr
			h := &webhookHandler{store: s}

			got := h.recordEntry(context.Background(), "U123", tc.text)

			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("recordEntry(%q) = %q, want substring %q", tc.text, got, tc.wantSubstr)
			}
			saved := len(s.entries["U123"]) > 0
			if saved != tc.wantSaved {
				t.Errorf("recordEntry(%q) saved = %v, want %v", tc.text, saved, tc.wantSaved)
			}
		})
	}
}

func sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testChannelSecret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestServeHTTP_ValidSignatureRecordsAndReplies(t *testing.T) {
	s := newFakeStore()
	r := &fakeReplier{}
	h := newWebhookHandler(testChannelSecret, r, s)

	body := []byte(`{
		"events": [{
			"type": "message",
			"replyToken": "reply-token-123",
			"source": {"type": "user", "userId": "U123"},
			"message": {"type": "text", "id": "1", "text": "-100 午餐"}
		}]
	}`)

	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(string(body)))
	req.Header.Set("X-Line-Signature", sign(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(s.entries["U123"]) != 1 {
		t.Fatalf("entries saved for U123 = %d, want 1", len(s.entries["U123"]))
	}
	if got := r.lastText(); !strings.Contains(got, "已記錄") {
		t.Fatalf("reply text = %q, want it to contain 已記錄", got)
	}
	if len(r.calls) != 1 || r.calls[0].ReplyToken != "reply-token-123" {
		t.Fatalf("reply not sent with expected reply token: %+v", r.calls)
	}
}

func TestServeHTTP_InvalidSignatureRejected(t *testing.T) {
	s := newFakeStore()
	r := &fakeReplier{}
	h := newWebhookHandler(testChannelSecret, r, s)

	body := []byte(`{"events": []}`)
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(string(body)))
	req.Header.Set("X-Line-Signature", "not-a-valid-signature")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if len(s.entries) != 0 {
		t.Fatalf("store should not be touched on invalid signature, got %v", s.entries)
	}
	if len(r.calls) != 0 {
		t.Fatalf("bot should not be called on invalid signature, got %d calls", len(r.calls))
	}
}
