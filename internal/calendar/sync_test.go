package calendar

import (
	"strings"
	"testing"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/jaredwarren/clock/internal/config"
)

func TestGenerateTickEvents_OneTime(t *testing.T) {
	const icsStr = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:evt1
DTSTART:20240615T100000Z
DTEND:20240615T110000Z
SUMMARY:One time event
END:VEVENT
END:VCALENDAR`

	cal, err := ics.ParseCalendar(strings.NewReader(icsStr))
	if err != nil {
		t.Fatalf("parse calendar: %v", err)
	}

	calCfg := config.CalendarConfig{
		OverridePastColor:    0x111111,
		OverridePresentColor: 0x222222,
		OverrideFutureColor:  0x333333,
		OverrideFutureBColor: 0x444444,
	}

	windowStart := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)

	events, err := generateTickEventsFromICalCalendar(
		cal,
		windowStart.UTC(),
		windowEnd.UTC(),
		windowStart.UTC().Add(-1*time.Nanosecond),
		windowEnd.UTC().Add(1*time.Nanosecond),
		calCfg,
	)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if !e.Start.Equal(time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start: %v", e.Start)
	}
	if !e.End.Equal(time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected end: %v", e.End)
	}
	if e.Repeat != config.RepeatNone {
		t.Fatalf("unexpected repeat: %q", e.Repeat)
	}
	if e.PastColorOverride != calCfg.OverridePastColor ||
		e.PresentColorOverride != calCfg.OverridePresentColor ||
		e.FutureColorOverride != calCfg.OverrideFutureColor ||
		e.FutureColorBOverride != calCfg.OverrideFutureBColor {
		t.Fatalf("expected override colors to be applied")
	}
}

func TestGenerateTickEvents_RRULEDailyCount(t *testing.T) {
	const icsStr = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:evt2
DTSTART:20240615T120000Z
DTEND:20240615T123000Z
RRULE:FREQ=DAILY;COUNT=3
SUMMARY:Recurring
END:VEVENT
END:VCALENDAR`

	cal, err := ics.ParseCalendar(strings.NewReader(icsStr))
	if err != nil {
		t.Fatalf("parse calendar: %v", err)
	}

	calCfg := config.CalendarConfig{
		OverridePastColor:    0xAAAAAA,
		OverridePresentColor: 0xBBBBBB,
		OverrideFutureColor:  0xCCCCCC,
		OverrideFutureBColor: 0xDDDDDD,
	}

	windowStart := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2024, 6, 18, 0, 0, 0, 0, time.UTC)

	events, err := generateTickEventsFromICalCalendar(
		cal,
		windowStart.UTC(),
		windowEnd.UTC(),
		windowStart.UTC().Add(-1*time.Nanosecond),
		windowEnd.UTC().Add(1*time.Nanosecond),
		calCfg,
	)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Sorted by Start.
	expectedStarts := []time.Time{
		time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 16, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 17, 12, 0, 0, 0, time.UTC),
	}
	for i := range expectedStarts {
		if !events[i].Start.Equal(expectedStarts[i]) {
			t.Fatalf("event[%d] unexpected start: %v", i, events[i].Start)
		}
		if !events[i].End.Equal(expectedStarts[i].Add(30 * time.Minute)) {
			t.Fatalf("event[%d] unexpected end: %v", i, events[i].End)
		}
	}
}

func TestGenerateTickEvents_OvernightSplitsByDay(t *testing.T) {
	const icsStr = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:evt3
DTSTART:20240615T230000Z
DTEND:20240616T010000Z
SUMMARY:Overnight
END:VEVENT
END:VCALENDAR`

	cal, err := ics.ParseCalendar(strings.NewReader(icsStr))
	if err != nil {
		t.Fatalf("parse calendar: %v", err)
	}

	calCfg := config.CalendarConfig{
		OverridePastColor:    0x010203,
		OverridePresentColor: 0x040506,
		OverrideFutureColor:  0x070809,
		OverrideFutureBColor: 0x0A0B0C,
	}

	windowStart := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC)

	events, err := generateTickEventsFromICalCalendar(
		cal,
		windowStart.UTC(),
		windowEnd.UTC(),
		windowStart.UTC().Add(-1*time.Nanosecond),
		windowEnd.UTC().Add(1*time.Nanosecond),
		calCfg,
	)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 split events, got %d", len(events))
	}

	// First segment: 23:00-00:00; Second: 00:00-01:00
	if !events[0].Start.Equal(time.Date(2024, 6, 15, 23, 0, 0, 0, time.UTC)) {
		t.Fatalf("segment[0] unexpected start: %v", events[0].Start)
	}
	if !events[0].End.Equal(time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("segment[0] unexpected end: %v", events[0].End)
	}

	if !events[1].Start.Equal(time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("segment[1] unexpected start: %v", events[1].Start)
	}
	if !events[1].End.Equal(time.Date(2024, 6, 16, 1, 0, 0, 0, time.UTC)) {
		t.Fatalf("segment[1] unexpected end: %v", events[1].End)
	}
}

