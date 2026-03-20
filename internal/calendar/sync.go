package calendar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/graham/rrule"
	"github.com/jaredwarren/clock/internal/config"
)

// SyncICalToTickEvents fetches + parses cfg.Calendar.ICalURL and converts matching VEVENT occurrences
// into concrete config.TickEvent instances. Manual cfg.Tick.Events are not preserved: callers should
// overwrite Tick.Events with the returned slice.
func SyncICalToTickEvents(ctx context.Context, cfg *config.Config, now time.Time) ([]config.TickEvent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	calCfg := cfg.Calendar
	if strings.TrimSpace(calCfg.ICalURL) == "" {
		return nil, nil
	}

	// Window is [now - lookback days, now + lookahead days] at the same clock time as `now`
	// (calendar-day offsets, not “start of day” to end of day).
	lb := calCfg.LookbackDays
	la := calCfg.LookaheadDays
	if lb < 0 {
		lb = 0
	}
	if la < 0 {
		la = 0
	}
	windowStart := now.AddDate(0, 0, -lb)
	windowEnd := now.AddDate(0, 0, la)
	if !windowEnd.After(windowStart) {
		// lookback=0 and lookahead=0 → start==end==now, so nothing overlaps except an event
		// active at this exact instant. Expand to defaults so feeds still produce events.
		lb, la = 1, 14
		windowStart = now.AddDate(0, 0, -lb)
		windowEnd = now.AddDate(0, 0, la)
	}

	// rrule.Iterator.Between excludes equality, so expand by 1ns on both ends.
	rruleAfter := windowStart.UTC().Add(-1 * time.Nanosecond)
	rruleBefore := windowEnd.UTC().Add(1 * time.Nanosecond)

	cal, err := ics.ParseCalendarFromUrl(calCfg.ICalURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("parse iCal from url: %w", err)
	}

	out, genErr := generateTickEventsFromICalCalendar(cal, windowStart.UTC(), windowEnd.UTC(), rruleAfter, rruleBefore, calCfg)
	if genErr != nil {
		return nil, genErr
	}

	return out, nil
}

func generateTickEventsFromICalCalendar(cal *ics.Calendar, windowStartUTC, windowEndUTC, rruleAfterUTC, rruleBeforeUTC time.Time, calCfg config.CalendarConfig) ([]config.TickEvent, error) {
	if cal == nil {
		return nil, fmt.Errorf("nil iCal calendar")
	}

	var out []config.TickEvent

	for _, vevent := range cal.Events() {
		if vevent == nil {
			continue
		}

		title := veventTitle(vevent)
		uid := veventUID(vevent)

		dtStart, err := vevent.GetStartAt()
		if err != nil {
			return nil, fmt.Errorf("VEVENT DTSTART parse: %w", err)
		}

		dtEnd, err := vevent.GetEndAt()
		if err != nil {
			// If DTEND is missing, try DURATION (ISO8601-like e.g. PT15M).
			if durProp := vevent.GetProperty(ics.ComponentPropertyDuration); durProp != nil {
				d, derr := rrule.ParseDuration(durProp.Value)
				if derr == nil {
					dtEnd = dtStart.Add(d)
				} else {
					return nil, fmt.Errorf("VEVENT DTEND parse: %w", err)
				}
			} else {
				return nil, fmt.Errorf("VEVENT DTEND parse: %w", err)
			}
		}

		if !dtEnd.After(dtStart) {
			// Zero/negative durations don't produce meaningful tick overrides.
			continue
		}

		rruleProp := vevent.GetProperty(ics.ComponentPropertyRrule)
		if rruleProp == nil || strings.TrimSpace(rruleProp.Value) == "" {
			// One-time event.
			if overlapsWindow(dtStart.UTC(), dtEnd.UTC(), windowStartUTC, windowEndUTC) {
				out = append(out, splitOccurrenceToDailyTickEvents(uid, title, dtStart.UTC(), dtEnd.UTC(), calCfg)...)
			}
			continue
		}

		// Recurring event. Expand RRULE occurrences into concrete one-time occurrences.
		rruleValue := strings.TrimSpace(rruleProp.Value)
		rruleValue = strings.TrimPrefix(rruleValue, "RRULE:")

		dtStartUTC := dtStart.UTC()
		dtStartLine := fmt.Sprintf("DTSTART;TZID=UTC:%s", dtStartUTC.Format("20060102T150405"))
		ruleText := dtStartLine + "\n" + "RRULE:" + rruleValue

		rule, err := rrule.Parse(ruleText)
		if err != nil {
			return nil, fmt.Errorf("parse RRULE for uid=%q: %w", uid, err)
		}

		iter := rule.Iterator().Between(rruleAfterUTC, rruleBeforeUTC)

		dur := dtEnd.Sub(dtStart)
		var occStart time.Time
		for iter.Step(&occStart) {
			occEnd := occStart.Add(dur)
			if !overlapsWindow(occStart, occEnd, windowStartUTC, windowEndUTC) {
				continue
			}
			out = append(out, splitOccurrenceToDailyTickEvents(uid, title, occStart, occEnd, calCfg)...)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Start.Equal(out[j].Start) {
			if out[i].End.Equal(out[j].End) {
				return out[i].ID < out[j].ID
			}
			return out[i].End.Before(out[j].End)
		}
		return out[i].Start.Before(out[j].Start)
	})

	return out, nil
}

func veventUID(e *ics.VEvent) string {
	// UniqueId is the stable UID across syncs; fall back to title if missing.
	if e == nil {
		return ""
	}
	if id := strings.TrimSpace(e.Id()); id != "" {
		return id
	}
	return ""
}

func veventTitle(e *ics.VEvent) string {
	if e == nil {
		return ""
	}
	if p := e.GetProperty(ics.ComponentPropertySummary); p != nil && strings.TrimSpace(p.Value) != "" {
		return ics.FromText(p.Value)
	}
	if id := strings.TrimSpace(e.Id()); id != "" {
		return id
	}
	return "iCal event"
}

func overlapsWindow(start, end, windowStart, windowEnd time.Time) bool {
	// We treat End as an exclusive boundary for overlap decisions (matches tick overlap math).
	return start.Before(windowEnd) && end.After(windowStart)
}

func splitOccurrenceToDailyTickEvents(uid, title string, occStartUTC, occEndUTC time.Time, calCfg config.CalendarConfig) []config.TickEvent {
	// Split events that cross midnight into per-day segments, so display overrides apply
	// across the entire span.
	if !occEndUTC.After(occStartUTC) {
		return nil
	}

	startDay := time.Date(occStartUTC.Year(), occStartUTC.Month(), occStartUTC.Day(), 0, 0, 0, 0, time.UTC)
	endDay := time.Date(occEndUTC.Year(), occEndUTC.Month(), occEndUTC.Day(), 0, 0, 0, 0, time.UTC)

	// If an event ends exactly at midnight, it belongs to the prior day only.
	if occEndUTC.Equal(endDay) {
		endDay = endDay.Add(-24 * time.Hour)
	}

	var out []config.TickEvent
	for day := startDay; !day.After(endDay); day = day.Add(24 * time.Hour) {
		dayStart := day
		dayEnd := day.Add(24 * time.Hour)

		segStart := maxTime(occStartUTC, dayStart)
		segEnd := minTime(occEndUTC, dayEnd)
		if !segEnd.After(segStart) {
			continue
		}

		id := stableICalEventID(uid, segStart, segEnd)
		out = append(out, config.TickEvent{
			ID:                   id,
			Title:                title,
			Start:                segStart,
			End:                  segEnd,
			Repeat:               config.RepeatNone,
			PastColorOverride:    calCfg.OverridePastColor,
			PresentColorOverride: calCfg.OverridePresentColor,
			FutureColorOverride:  calCfg.OverrideFutureColor,
			FutureColorBOverride: calCfg.OverrideFutureBColor,
		})
	}

	return out
}

func stableICalEventID(uid string, segStart, segEnd time.Time) string {
	seed := fmt.Sprintf("%s|%d|%d", uid, segStart.UnixNano(), segEnd.UnixNano())
	h := sha256.Sum256([]byte(seed))
	// Shorten so IDs remain human-friendly-ish.
	return "ical_" + hex.EncodeToString(h[:8])
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

