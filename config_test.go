package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigDefaultsTTSModel(t *testing.T) {
	cfg := loadTestConfig(t, `
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
users:
  - name: "bob"
`)
	if got, want := cfg.TTSModel(), defaultTTSModel; got != want {
		t.Fatalf("TTSModel = %q, want %q", got, want)
	}
}

func TestLoadConfigCustomTTSModel(t *testing.T) {
	cfg := loadTestConfig(t, `
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
tts:
  model: "en_US-ryan-medium"
users:
  - name: "bob"
`)
	if got, want := cfg.TTSModel(), "en_US-ryan-medium"; got != want {
		t.Fatalf("TTSModel = %q, want %q", got, want)
	}
}

func TestLoadConfigValidPerDayOverride(t *testing.T) {
	cfg := loadTestConfig(t, `
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
users:
  - name: "bob"
    allowed_hours:
      start: 8
      end: 18
    allowed_hours_by_day:
      saturday:
        start: 7
        end: 20
`)
	u := cfg.getUser("bob")
	if u == nil {
		t.Fatal("getUser(bob) = nil")
	}
	sat, ok := u.AllowedHoursByDay["saturday"]
	if !ok {
		t.Fatal("saturday override missing")
	}
	if (sat != AllowedHours{Start: 7, End: 20}) {
		t.Fatalf("saturday = %+v, want {7 0 20 0}", sat)
	}
}

func TestLoadConfigUnknownDayKeyErrors(t *testing.T) {
	err := loadTestConfigErr(t, `
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
users:
  - name: "bob"
    allowed_hours_by_day:
      saterday:
        start: 7
        end: 20
`)
	if err == nil {
		t.Fatal("expected error for unknown day key, got nil")
	}
}

func TestLoadConfigOverrideStartNotBeforeEndErrors(t *testing.T) {
	err := loadTestConfigErr(t, `
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
users:
  - name: "bob"
    allowed_hours_by_day:
      saturday:
        start: 20
        end: 7
`)
	if err == nil {
		t.Fatal("expected error for start >= end override, got nil")
	}
}

func TestLoadConfigDefaultFallbackOnlyWhenNoDefaultAndNoOverrides(t *testing.T) {
	cfg := loadTestConfig(t, `
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
users:
  - name: "bob"
`)
	u := cfg.getUser("bob")
	if (u.AllowedHours != AllowedHours{Start: 8, End: 18}) {
		t.Fatalf("default fallback = %+v, want {8 0 18 0}", u.AllowedHours)
	}
}

func TestLoadConfigOverridesDoNotTriggerDefaultFallback(t *testing.T) {
	// allowed_hours is left unset (0/0) but a midnight override is present;
	// the override must be honored literally and the default must NOT be
	// rewritten to 8-18 by the fallback.
	cfg := loadTestConfig(t, `
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
users:
  - name: "bob"
    allowed_hours_by_day:
      friday:
        start: 0
        end: 8
`)
	u := cfg.getUser("bob")
	if (u.AllowedHoursByDay["friday"] != AllowedHours{Start: 0, End: 8}) {
		t.Fatalf("friday override = %+v, want {0 0 8 0}", u.AllowedHoursByDay["friday"])
	}
	// The top-level default still gets its own fallback (unrelated to overrides).
	if (u.AllowedHours != AllowedHours{Start: 8, End: 18}) {
		t.Fatalf("default = %+v, want {8 0 18 0}", u.AllowedHours)
	}
}

func TestShowConfigRendersCompiledDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
machine_name: "test"
telegram:
  bot_token: "token"
  allowed_chat_ids: [123]
users:
  - name: "bob"
`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	out, err := showConfig(path)
	if err != nil {
		t.Fatalf("showConfig: %v", err)
	}
	// Defaults applied by loadConfig should appear in the rendered output.
	for _, want := range []string{
		"listen_addr: 127.0.0.1:3847",
		"daily_limit_minutes: 300",
		"thresholds:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("showConfig output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func loadTestConfig(t *testing.T, content string) *Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	return cfg
}

func loadTestConfigErr(t *testing.T, content string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := loadConfig(path)
	return err
}
