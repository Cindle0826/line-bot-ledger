package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
	"github.com/cindle0826/line-bot-ledger/internal/store"
)

// pusher is the slice of the messaging_api client this handler needs.
// Narrowed like services/webhook's replier so tests can inject a fake.
type pusher interface {
	PushMessage(*messaging_api.PushMessageRequest, string) (*messaging_api.PushMessageResponse, error)
}

// summaryHandler serves the endpoint Cloud Scheduler and Cloud Run's health
// check hit.
type summaryHandler struct {
	store  store.Store
	pusher pusher
	now    func() time.Time
}

func newSummaryHandler(s store.Store, p pusher) *summaryHandler {
	return &summaryHandler{store: s, pusher: p, now: time.Now}
}

// job is one frequency's summary window for a single run.
type job struct {
	frequency ledger.Frequency
	label     string
	from, to  time.Time
}

// run fans out over every frequency due today, looks up who's subscribed to
// each, and pushes each of them their own summary for that window. Cloud
// Scheduler calls this once a day — weekly/monthly aren't separate
// schedules, they're conditions checked on the same daily trigger.
func (h *summaryHandler) run(w http.ResponseWriter, r *http.Request) {
	now := h.now()
	jobs := []job{dailyJob(now)}
	if ledger.IsWeeklyTriggerDay(now) {
		jobs = append(jobs, weeklyJob(now))
	}
	if ledger.IsMonthlyTriggerDay(now) {
		jobs = append(jobs, monthlyJob(now))
	}

	ok := true
	for _, j := range jobs {
		if err := h.runJob(r.Context(), j); err != nil {
			slog.Error("summary: job failed", "frequency", j.frequency, "error", err)
			ok = false
		}
	}
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// runJob pushes to every subscriber of j.frequency, continuing past a
// single user's failure (a blocked bot or a bad push token shouldn't stop
// everyone else's summary) but reporting that a failure happened.
func (h *summaryHandler) runJob(ctx context.Context, j job) error {
	userIDs, err := h.store.ListSubscribers(ctx, j.frequency)
	if err != nil {
		return fmt.Errorf("list subscribers: %w", err)
	}

	var firstErr error
	for _, uid := range userIDs {
		if err := h.pushOne(ctx, uid, j); err != nil {
			slog.Error("summary: push failed", "user", uid, "frequency", j.frequency, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (h *summaryHandler) pushOne(ctx context.Context, uid string, j job) error {
	entries, err := h.store.ListEntries(ctx, uid, j.from, j.to)
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}
	text := formatSummary(j.label, ledger.Summarize(entries))
	_, err = h.pusher.PushMessage(&messaging_api.PushMessageRequest{
		To:       uid,
		Messages: []messaging_api.MessageInterface{messaging_api.TextMessage{Text: text}},
	}, "")
	return err
}

func dailyJob(now time.Time) job {
	from, to := ledger.DayRange(now)
	return job{ledger.FrequencyDaily, fmt.Sprintf("今日（%s）", from.Format("2006-01-02")), from, to}
}

func weeklyJob(now time.Time) job {
	from, to := ledger.WeekRange(now)
	return job{ledger.FrequencyWeekly, fmt.Sprintf("本週（%s ～ %s）", from.Format("2006-01-02"), to.Add(-24*time.Hour).Format("2006-01-02")), from, to}
}

func monthlyJob(now time.Time) job {
	from, to := ledger.MonthRange(now)
	return job{ledger.FrequencyMonthly, from.Format("2006年01月"), from, to}
}

// formatSummary renders a Summary as the push message text.
func formatSummary(label string, s ledger.Summary) string {
	if len(s.Categories) == 0 {
		return fmt.Sprintf("🦉 記帳總結（%s）\n\n這段期間還沒有記帳紀錄", label)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "🦉 記帳總結（%s）\n\n", label)
	for _, c := range s.Categories {
		fmt.Fprintf(&b, "%s：%s\n", c.Category, formatAmount(c.Total))
	}
	fmt.Fprintf(&b, "\n總計：%s", formatAmount(s.Total))
	return b.String()
}

func formatAmount(amount float64) string {
	if amount > 0 {
		return fmt.Sprintf("+%.0f", amount)
	}
	return fmt.Sprintf("%.0f", amount)
}
