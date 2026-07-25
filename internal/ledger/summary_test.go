package ledger

import (
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	entries := []Entry{
		{Amount: -100, Category: "午餐"},
		{Amount: -50, Category: "交通"},
		{Amount: -80, Category: "午餐"},
		{Amount: 5000, Category: "薪水"},
	}

	got := Summarize(entries)

	want := Summary{
		Categories: []CategoryTotal{
			{Category: "午餐", Total: -180},
			{Category: "交通", Total: -50},
			{Category: "薪水", Total: 5000},
		},
		Total: 4770,
	}

	if got.Total != want.Total {
		t.Errorf("Total = %v, want %v", got.Total, want.Total)
	}
	if len(got.Categories) != len(want.Categories) {
		t.Fatalf("len(Categories) = %d, want %d", len(got.Categories), len(want.Categories))
	}
	for i, c := range want.Categories {
		if got.Categories[i] != c {
			t.Errorf("Categories[%d] = %+v, want %+v", i, got.Categories[i], c)
		}
	}
}

func TestSummarize_Empty(t *testing.T) {
	got := Summarize(nil)
	if got.Total != 0 || len(got.Categories) != 0 {
		t.Errorf("Summarize(nil) = %+v, want zero value", got)
	}
}

func TestDayRange(t *testing.T) {
	// 2026-07-25 23:30 UTC = 2026-07-26 07:30 Taipei, so "today" in Taipei
	// is the 26th, not the 25th.
	in := time.Date(2026, 7, 25, 23, 30, 0, 0, time.UTC)
	from, to := DayRange(in)

	wantFrom := time.Date(2026, 7, 26, 0, 0, 0, 0, Taipei)
	wantTo := wantFrom.Add(24 * time.Hour)

	if !from.Equal(wantFrom) {
		t.Errorf("from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Errorf("to = %v, want %v", to, wantTo)
	}
}

func TestIsWeeklyTriggerDay(t *testing.T) {
	monday := time.Date(2026, time.January, 5, 21, 0, 0, 0, Taipei)
	tuesday := time.Date(2026, time.January, 6, 21, 0, 0, 0, Taipei)
	if !IsWeeklyTriggerDay(monday) {
		t.Error("Monday should be a weekly trigger day")
	}
	if IsWeeklyTriggerDay(tuesday) {
		t.Error("Tuesday should not be a weekly trigger day")
	}
}

func TestIsMonthlyTriggerDay(t *testing.T) {
	first := time.Date(2026, time.February, 1, 21, 0, 0, 0, Taipei)
	second := time.Date(2026, time.February, 2, 21, 0, 0, 0, Taipei)
	if !IsMonthlyTriggerDay(first) {
		t.Error("the 1st should be a monthly trigger day")
	}
	if IsMonthlyTriggerDay(second) {
		t.Error("the 2nd should not be a monthly trigger day")
	}
}

func TestWeekRange(t *testing.T) {
	monday := time.Date(2026, time.January, 5, 21, 0, 0, 0, Taipei)
	from, to := WeekRange(monday)

	wantTo := time.Date(2026, time.January, 5, 0, 0, 0, 0, Taipei)
	wantFrom := wantTo.Add(-7 * 24 * time.Hour)

	if !from.Equal(wantFrom) {
		t.Errorf("from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Errorf("to = %v, want %v", to, wantTo)
	}
}

func TestMonthRange(t *testing.T) {
	firstOfFeb := time.Date(2026, time.February, 1, 21, 0, 0, 0, Taipei)
	from, to := MonthRange(firstOfFeb)

	wantTo := time.Date(2026, time.February, 1, 0, 0, 0, 0, Taipei)
	wantFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, Taipei)

	if !from.Equal(wantFrom) {
		t.Errorf("from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Errorf("to = %v, want %v", to, wantTo)
	}
}
