package ledger

import "time"

// Taipei is fixed-offset (no DST in Taiwan), so it works in distroless
// containers that don't ship the IANA tzdata database — time.LoadLocation
// would fail there.
var Taipei = time.FixedZone("Asia/Taipei", 8*60*60)

// DayRange returns the [from, to) bounds of t's calendar day in Taipei
// time, for querying "today's" entries.
func DayRange(t time.Time) (from, to time.Time) {
	t = t.In(Taipei)
	from = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, Taipei)
	return from, from.Add(24 * time.Hour)
}

// Frequency is a services/summary subscription cadence, stored per-user in
// Firestore (users/{lineUserID}.subscriptions) and matched against
// IsWeeklyTriggerDay/IsMonthlyTriggerDay on each daily run.
type Frequency string

const (
	FrequencyDaily   Frequency = "daily"
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
)

// Valid reports whether f is one of the known frequencies — Go has no enum
// reflection, so a switch is the idiomatic stand-in.
func (f Frequency) Valid() bool {
	switch f {
	case FrequencyDaily, FrequencyWeekly, FrequencyMonthly:
		return true
	default:
		return false
	}
}

// IsWeeklyTriggerDay reports whether t is the day services/summary sends
// weekly summaries. Monday, so WeekRange(t) lines up with a clean Mon-Sun
// week.
func IsWeeklyTriggerDay(t time.Time) bool {
	return t.In(Taipei).Weekday() == time.Monday
}

// IsMonthlyTriggerDay reports whether t is the day services/summary sends
// monthly summaries. The 1st, so MonthRange(t) covers the just-finished
// previous month.
func IsMonthlyTriggerDay(t time.Time) bool {
	return t.In(Taipei).Day() == 1
}

// WeekRange returns the [from, to) bounds of the 7 days immediately before
// t's calendar day — the week a trigger on IsWeeklyTriggerDay(t) summarizes.
func WeekRange(t time.Time) (from, to time.Time) {
	to, _ = DayRange(t)
	from = to.Add(-7 * 24 * time.Hour)
	return from, to
}

// MonthRange returns the [from, to) bounds of the calendar month before
// t's — the month a trigger on IsMonthlyTriggerDay(t) summarizes.
func MonthRange(t time.Time) (from, to time.Time) {
	t = t.In(Taipei)
	to = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, Taipei)
	from = to.AddDate(0, -1, 0)
	return from, to
}

// CategoryTotal is one category's net total within a summary period.
type CategoryTotal struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// Summary aggregates entries by category, in first-seen order, plus a
// grand total across all of them.
type Summary struct {
	Categories []CategoryTotal `json:"categories"`
	Total      float64         `json:"total"`
}

// Summarize groups entries by category and sums each. Order matches each
// category's first appearance in entries, so output is stable given a
// stable input order (entries are normally already sorted by createdAt).
func Summarize(entries []Entry) Summary {
	order := make([]string, 0, len(entries))
	totals := make(map[string]float64, len(entries))
	var grand float64

	for _, e := range entries {
		if _, seen := totals[e.Category]; !seen {
			order = append(order, e.Category)
		}
		totals[e.Category] += e.Amount
		grand += e.Amount
	}

	categories := make([]CategoryTotal, len(order))
	for i, c := range order {
		categories[i] = CategoryTotal{Category: c, Total: totals[c]}
	}
	return Summary{Categories: categories, Total: grand}
}
