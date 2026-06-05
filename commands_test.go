package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// newHoursCmd builds an AdminCommands over a single user whose config mutations
// are persisted to a throwaway path inside t.TempDir().
func newHoursCmd(t *testing.T, u UserConfig) *AdminCommands {
	t.Helper()
	cmd := NewAdminCommands(&Config{Users: []UserConfig{u}}, nil)
	cmd.savePath = filepath.Join(t.TempDir(), "config.yaml")
	return cmd
}

func TestHoursSetsDefault(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{Name: "bob", AllowedHours: AllowedHours{Start: 8, End: 18}})

	out, err := cmd.Hours([]string{"bob", "9-17"})
	if err != nil {
		t.Fatalf("Hours set default: %v", err)
	}
	if !strings.Contains(out, "9am - 5pm") {
		t.Fatalf("Hours set default = %q, want it to mention 9am - 5pm", out)
	}
	got := cmd.cfg.getUser("bob").AllowedHours
	want := AllowedHours{Start: 9, End: 17}
	if got != want {
		t.Fatalf("AllowedHours = %+v, want %+v", got, want)
	}
}

func TestHoursSetsDayOverride(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{Name: "bob", AllowedHours: AllowedHours{Start: 8, End: 18}})

	if _, err := cmd.Hours([]string{"bob", "saturday", "7-20"}); err != nil {
		t.Fatalf("Hours set override: %v", err)
	}
	got, ok := cmd.cfg.getUser("bob").AllowedHoursByDay["saturday"]
	if !ok {
		t.Fatal("saturday override not set")
	}
	want := AllowedHours{Start: 7, End: 20}
	if got != want {
		t.Fatalf("saturday override = %+v, want %+v", got, want)
	}
}

func TestHoursReplacesExistingOverride(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{
		Name:              "bob",
		AllowedHours:      AllowedHours{Start: 8, End: 18},
		AllowedHoursByDay: map[string]AllowedHours{"saturday": {Start: 7, End: 20}},
	})

	if _, err := cmd.Hours([]string{"bob", "saturday", "6-22"}); err != nil {
		t.Fatalf("Hours replace override: %v", err)
	}
	got := cmd.cfg.getUser("bob").AllowedHoursByDay["saturday"]
	want := AllowedHours{Start: 6, End: 22}
	if got != want {
		t.Fatalf("saturday override = %+v, want %+v", got, want)
	}
}

func TestHoursClearsOverride(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{
		Name:              "bob",
		AllowedHours:      AllowedHours{Start: 8, End: 18},
		AllowedHoursByDay: map[string]AllowedHours{"saturday": {Start: 7, End: 20}},
	})

	out, err := cmd.Hours([]string{"bob", "saturday", "clear"})
	if err != nil {
		t.Fatalf("Hours clear override: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "cleared") {
		t.Fatalf("Hours clear override = %q, want it to confirm the clear", out)
	}
	if _, ok := cmd.cfg.getUser("bob").AllowedHoursByDay["saturday"]; ok {
		t.Fatal("saturday override still present after clear")
	}
}

func TestHoursClearMissingOverride(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{Name: "bob", AllowedHours: AllowedHours{Start: 8, End: 18}})

	out, err := cmd.Hours([]string{"bob", "saturday", "clear"})
	if err != nil {
		t.Fatalf("Hours clear missing override: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "no saturday override") {
		t.Fatalf("Hours clear missing override = %q, want it to report no override", out)
	}
}

func TestHoursShowsSchedule(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{
		Name:              "bob",
		AllowedHours:      AllowedHours{Start: 8, End: 18},
		AllowedHoursByDay: map[string]AllowedHours{"saturday": {Start: 7, End: 20}},
	})

	out, err := cmd.Hours([]string{"bob"})
	if err != nil {
		t.Fatalf("Hours show: %v", err)
	}
	if !strings.Contains(out, "default: 8am - 6pm") || !strings.Contains(out, "saturday: 7am - 8pm") {
		t.Fatalf("Hours show = %q, want default and saturday lines", out)
	}
}

func TestHoursUnknownDayErrors(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{Name: "bob", AllowedHours: AllowedHours{Start: 8, End: 18}})

	if _, err := cmd.Hours([]string{"bob", "saterday", "7-20"}); err == nil {
		t.Fatal("Hours with unknown day succeeded, want error")
	}
}

func TestHoursBadRangeErrors(t *testing.T) {
	cmd := newHoursCmd(t, UserConfig{Name: "bob", AllowedHours: AllowedHours{Start: 8, End: 18}})

	if _, err := cmd.Hours([]string{"bob", "saturday", "20-7"}); err == nil {
		t.Fatal("Hours with start after end succeeded, want error")
	}
	if _, err := cmd.Hours([]string{"bob", "nonsense"}); err == nil {
		t.Fatal("Hours with bad default range succeeded, want error")
	}
}
