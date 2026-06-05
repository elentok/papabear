package main

import (
	"testing"
	"time"
)

func TestParseWeekday(t *testing.T) {
	for name, want := range weekdayByName {
		got, err := parseWeekday(name)
		if err != nil {
			t.Errorf("parseWeekday(%q) returned error: %v", name, err)
		}
		if got != want {
			t.Errorf("parseWeekday(%q) = %v, want %v", name, got, want)
		}
	}
	if _, err := parseWeekday("saterday"); err == nil {
		t.Error("parseWeekday(\"saterday\") expected error, got nil")
	}
	if _, err := parseWeekday("Monday"); err == nil {
		t.Error("parseWeekday(\"Monday\") expected error for non-lowercase, got nil")
	}
}

func TestEffectiveAllowedHours(t *testing.T) {
	def := AllowedHours{Start: 8, End: 18}

	t.Run("no overrides falls back to default every weekday", func(t *testing.T) {
		for wd := time.Sunday; wd <= time.Saturday; wd++ {
			if got := effectiveAllowedHours(def, nil, wd); got != def {
				t.Errorf("weekday %v: got %+v, want default %+v", wd, got, def)
			}
		}
	})

	t.Run("single override differs only on that day", func(t *testing.T) {
		sat := AllowedHours{Start: 7, End: 20}
		byDay := map[string]AllowedHours{"saturday": sat}
		for wd := time.Sunday; wd <= time.Saturday; wd++ {
			got := effectiveAllowedHours(def, byDay, wd)
			want := def
			if wd == time.Saturday {
				want = sat
			}
			if got != want {
				t.Errorf("weekday %v: got %+v, want %+v", wd, got, want)
			}
		}
	})

	t.Run("multiple overrides", func(t *testing.T) {
		sat := AllowedHours{Start: 7, End: 20}
		tue := AllowedHours{Start: 7, End: 18}
		byDay := map[string]AllowedHours{"saturday": sat, "tuesday": tue}
		if got := effectiveAllowedHours(def, byDay, time.Saturday); got != sat {
			t.Errorf("saturday: got %+v, want %+v", got, sat)
		}
		if got := effectiveAllowedHours(def, byDay, time.Tuesday); got != tue {
			t.Errorf("tuesday: got %+v, want %+v", got, tue)
		}
		if got := effectiveAllowedHours(def, byDay, time.Monday); got != def {
			t.Errorf("monday: got %+v, want default %+v", got, def)
		}
	})

	t.Run("midnight-boundary override honored literally", func(t *testing.T) {
		night := AllowedHours{Start: 0, End: 8}
		byDay := map[string]AllowedHours{"friday": night}
		if got := effectiveAllowedHours(def, byDay, time.Friday); got != night {
			t.Errorf("friday: got %+v, want %+v", got, night)
		}
	})
}

func TestFormatSchedule(t *testing.T) {
	def := AllowedHours{Start: 8, End: 18}
	sat := AllowedHours{Start: 7, End: 20}
	tue := AllowedHours{Start: 7, End: 18}

	tests := []struct {
		name  string
		byDay map[string]AllowedHours
		today time.Weekday
		want  string
	}{
		{
			name:  "default only, marker on default",
			byDay: nil,
			today: time.Monday,
			want:  "  default: 8am - 6pm (today)",
		},
		{
			name:  "one override, today is a non-override day -> marker on default",
			byDay: map[string]AllowedHours{"saturday": sat},
			today: time.Monday,
			want:  "  default: 8am - 6pm (today)\n  saturday: 7am - 8pm",
		},
		{
			name:  "one override, today is the override day -> marker on override line",
			byDay: map[string]AllowedHours{"saturday": sat},
			today: time.Saturday,
			want:  "  default: 8am - 6pm\n  saturday: 7am - 8pm (today)",
		},
		{
			name:  "multiple overrides render Monday-first",
			byDay: map[string]AllowedHours{"saturday": sat, "tuesday": tue},
			today: time.Tuesday,
			want:  "  default: 8am - 6pm\n  tuesday: 7am - 6pm (today)\n  saturday: 7am - 8pm",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatSchedule(def, tc.byDay, tc.today); got != tc.want {
				t.Errorf("formatSchedule:\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
		})
	}
}
