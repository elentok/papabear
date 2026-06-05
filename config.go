package main

import (
	"fmt"
	"os"
	"syscall"

	"gopkg.in/yaml.v3"
)

const configPath = "/etc/screentimectl/config.yaml"

type Config struct {
	MachineName   string             `yaml:"machine_name"`
	Telegram      TelegramConfig     `yaml:"telegram"`
	Server        ServerConfig       `yaml:"server"`
	Users         []UserConfig       `yaml:"users"`
	Notifications NotificationConfig `yaml:"notifications"`
	TTS           TTSConfig          `yaml:"tts"`
}

type TelegramConfig struct {
	BotToken       string  `yaml:"bot_token"`
	AllowedChatIDs []int64 `yaml:"allowed_chat_ids"`
}

type ServerConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

type UserConfig struct {
	Name              string                  `yaml:"name"`
	DailyLimitMinutes int                     `yaml:"daily_limit_minutes"`
	AllowedHours      AllowedHours            `yaml:"allowed_hours"`
	AllowedHoursByDay map[string]AllowedHours `yaml:"allowed_hours_by_day,omitempty"`
}

type AllowedHours struct {
	Start       int `yaml:"start"`
	StartMinute int `yaml:"start_minute,omitempty"`
	End         int `yaml:"end"`
	EndMinute   int `yaml:"end_minute,omitempty"`
}

type NotificationConfig struct {
	Thresholds []int `yaml:"thresholds"`
}

type TTSConfig struct {
	Model string `yaml:"model"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Telegram.BotToken == "" {
		return nil, fmt.Errorf("telegram.bot_token is required")
	}
	if len(cfg.Telegram.AllowedChatIDs) == 0 {
		return nil, fmt.Errorf("telegram.allowed_chat_ids must have at least one entry")
	}
	if len(cfg.Users) == 0 {
		return nil, fmt.Errorf("users must have at least one entry")
	}
	if cfg.Server.ListenAddr == "" {
		cfg.Server.ListenAddr = "127.0.0.1:3847"
	}
	for i := range cfg.Users {
		if cfg.Users[i].DailyLimitMinutes == 0 {
			cfg.Users[i].DailyLimitMinutes = 300
		}
		if cfg.Users[i].AllowedHours.Start == 0 && cfg.Users[i].AllowedHours.End == 0 {
			cfg.Users[i].AllowedHours = AllowedHours{Start: 8, End: 18}
		}
		for day, oh := range cfg.Users[i].AllowedHoursByDay {
			if _, err := parseWeekday(day); err != nil {
				return nil, fmt.Errorf("user %q: allowed_hours_by_day: %w", cfg.Users[i].Name, err)
			}
			if oh.Start*60+oh.StartMinute >= oh.End*60+oh.EndMinute {
				return nil, fmt.Errorf("user %q: allowed_hours_by_day[%s]: start must be before end", cfg.Users[i].Name, day)
			}
		}
	}
	if len(cfg.Notifications.Thresholds) == 0 {
		cfg.Notifications.Thresholds = []int{30, 15, 5, 1}
	}
	if cfg.TTS.Model == "" {
		cfg.TTS.Model = defaultTTSModel
	}

	return &cfg, nil
}

func (c *Config) TTSModel() string {
	if c.TTS.Model != "" {
		return c.TTS.Model
	}
	return defaultTTSModel
}

func (c *Config) isAllowedChat(chatID int64) bool {
	for _, id := range c.Telegram.AllowedChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

func (c *Config) isValidUser(name string) bool {
	for _, u := range c.Users {
		if u.Name == name {
			return true
		}
	}
	return false
}

func (c *Config) getUser(name string) *UserConfig {
	for i := range c.Users {
		if c.Users[i].Name == name {
			return &c.Users[i]
		}
	}
	return nil
}

// save writes the config to disk atomically.
func (c *Config) save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0640); err != nil {
		return err
	}
	// Preserve ownership of the original so the daemon can still read it
	// after the rename (the tmp is created by root via sudo).
	if fi, err := os.Stat(path); err == nil {
		if st, ok := fi.Sys().(*syscall.Stat_t); ok {
			_ = os.Chown(tmp, int(st.Uid), int(st.Gid))
		}
	}
	return os.Rename(tmp, path)
}

// Reload re-reads the config from disk and updates this object in place.
// The caller's pointer remains valid, so the HTTP server and session manager
// both see the new values without pointer swaps.
func (c *Config) Reload(path string) error {
	fresh, err := loadConfig(path)
	if err != nil {
		return err
	}
	*c = *fresh
	return nil
}
