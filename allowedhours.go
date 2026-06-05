package main

import (
	"fmt"
	"strings"
	"time"
)

// weekdayByName maps lowercase full day names to their time.Weekday.
var weekdayByName = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// parseWeekday maps a lowercase full day name (monday..sunday) to its
// time.Weekday, returning an error for any unrecognized word.
func parseWeekday(name string) (time.Weekday, error) {
	wd, ok := weekdayByName[name]
	if !ok {
		return 0, fmt.Errorf("unknown day: %q", name)
	}
	return wd, nil
}

// weekdayName returns the lowercase full name of a weekday (the key form used
// in allowed_hours_by_day).
func weekdayName(wd time.Weekday) string {
	return strings.ToLower(wd.String())
}

// effectiveAllowedHours resolves the Effective Allowed Hours for a given
// weekday: the Per-Day Override for that weekday if one is present, otherwise
// the Default Allowed Hours. Pure: no clock, no I/O.
func effectiveAllowedHours(def AllowedHours, byDay map[string]AllowedHours, wd time.Weekday) AllowedHours {
	if oh, ok := byDay[weekdayName(wd)]; ok {
		return oh
	}
	return def
}

// effectiveAllowedHoursNow resolves today's Effective Allowed Hours for the
// user, reading the local clock for the current weekday.
func (u UserConfig) effectiveAllowedHoursNow() AllowedHours {
	return effectiveAllowedHours(u.AllowedHours, u.AllowedHoursByDay, time.Now().Weekday())
}

// weekdayDisplayOrder lists weekdays Monday-first for stable schedule rendering.
var weekdayDisplayOrder = []time.Weekday{
	time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
	time.Friday, time.Saturday, time.Sunday,
}

// formatSchedule renders the full weekly Allowed Hours schedule: an indented
// "default" line plus one line per Per-Day Override (Monday-first), with
// "(today)" appended to whichever line is in force for the given weekday — the
// default line carries the marker when today has no override. Pure: the weekday
// is passed in. Callers add their own header.
func formatSchedule(def AllowedHours, byDay map[string]AllowedHours, today time.Weekday) string {
	_, todayHasOverride := byDay[weekdayName(today)]

	lines := []string{formatScheduleLine("default", def, !todayHasOverride)}
	for _, wd := range weekdayDisplayOrder {
		if oh, ok := byDay[weekdayName(wd)]; ok {
			lines = append(lines, formatScheduleLine(weekdayName(wd), oh, wd == today))
		}
	}
	return strings.Join(lines, "\n")
}

func formatScheduleLine(label string, h AllowedHours, isToday bool) string {
	line := fmt.Sprintf("  %s: %s - %s", label,
		formatHour(h.Start, h.StartMinute), formatHour(h.End, h.EndMinute))
	if isToday {
		line += " (today)"
	}
	return line
}
