package main

import (
	"fmt"
	"strings"
	"time"
)

type AdminCommands struct {
	cfg *Config
	mgr *SessionManager
	// savePath is where config mutations are persisted; defaults to configPath
	// and is overridable in tests.
	savePath string
}

func NewAdminCommands(cfg *Config, mgr *SessionManager) *AdminCommands {
	return &AdminCommands{cfg: cfg, mgr: mgr, savePath: configPath}
}

func (c *AdminCommands) Give(args []string) (string, error) {
	user, durStr, err := c.parseUserDuration(args)
	if err != nil {
		return "", fmt.Errorf("usage: give [user] {duration}: %w", err)
	}
	minutes, err := parseDurationMinutes(durStr)
	if err != nil {
		return "", fmt.Errorf("invalid duration: %w", err)
	}
	ut, err := c.mgr.AddTime(user, minutes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s now has %s remaining", capitalize(user), ut.RemainingStr()), nil
}

func (c *AdminCommands) Lock(args []string) (string, error) {
	user, durStr, err := c.parseUserOptionalDuration(args)
	if err != nil {
		return "", fmt.Errorf("usage: lock [user] [duration]: %w", err)
	}

	minutes := 0
	if durStr != "" {
		minutes, err = parseDurationMinutes(durStr)
		if err != nil {
			return "", fmt.Errorf("invalid duration: %w", err)
		}
	}

	if minutes == 0 {
		if err := c.mgr.LockUser(user); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s has been locked out", capitalize(user)), nil
	}

	ut, err := c.mgr.SetTime(user, minutes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s now has %s remaining", capitalize(user), ut.RemainingStr()), nil
}

func (c *AdminCommands) Unlock(args []string) (string, error) {
	user, durStr, err := c.parseUserDuration(args)
	if err != nil {
		return "", fmt.Errorf("usage: unlock [user] {duration}: %w", err)
	}
	minutes, err := parseDurationMinutes(durStr)
	if err != nil {
		return "", fmt.Errorf("invalid duration: %w", err)
	}
	ut, err := c.mgr.SetTime(user, minutes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s now has %s remaining", capitalize(user), ut.RemainingStr()), nil
}

func (c *AdminCommands) Status(args []string) (string, error) {
	user, err := c.parseOptionalUser(args)
	if err != nil {
		return "", fmt.Errorf("usage: status [user]: %w", err)
	}
	return c.StatusForUser(user)
}

func (c *AdminCommands) StatusForUser(user string) (string, error) {
	if _, err := c.requireValidUser(user); err != nil {
		return "", err
	}
	ut, err := c.mgr.GetUserTime(user)
	if err != nil {
		return "", err
	}

	u := c.cfg.getUser(user)
	schedule := formatSchedule(u.AllowedHours, u.AllowedHoursByDay, time.Now().Weekday())
	text := fmt.Sprintf("%s has %s remaining (used %s today)\nAllowed hours:\n%s\nSession: %s\nAccount: %s",
		capitalize(user), ut.RemainingStr(), ut.UsedStr(),
		schedule,
		ut.SessionStatus, ut.AccountStatus)

	if c.mgr.actLog != nil {
		entries, err := c.mgr.actLog.ReadDay(user, today())
		if err == nil && len(entries) > 0 {
			text += "\n\nToday:\n" + FormatTimeline(entries)
		}
	}

	return text, nil
}

func (c *AdminCommands) StatusSummaryForUser(user string, compact bool) (string, error) {
	if _, err := c.requireValidUser(user); err != nil {
		return "", err
	}
	ut, err := c.mgr.GetUserTime(user)
	if err != nil {
		return "", err
	}
	return formatStatusSummary(ut, compact), nil
}

const hoursUsage = "hours [user] | hours [user] start-end | hours [user] <day> start-end | hours [user] <day> clear"

func (c *AdminCommands) Hours(args []string) (string, error) {
	user, rest, err := c.splitOptionalUser(args)
	if err != nil {
		return "", fmt.Errorf("usage: %s: %w", hoursUsage, err)
	}
	u := c.cfg.getUser(user)

	switch len(rest) {
	case 0:
		// Show the full schedule.
		return fmt.Sprintf("Allowed hours for %s:\n%s",
			capitalize(user),
			formatSchedule(u.AllowedHours, u.AllowedHoursByDay, time.Now().Weekday())), nil

	case 1:
		// Set the Default Allowed Hours.
		ah, err := parseHoursRange(rest[0])
		if err != nil {
			return "", fmt.Errorf("invalid hours: %w", err)
		}
		u.AllowedHours = ah
		if err := c.cfg.save(c.savePath); err != nil {
			return "", fmt.Errorf("failed to save config: %w", err)
		}
		return fmt.Sprintf("Updated allowed hours for %s: %s - %s",
			capitalize(user),
			formatHour(ah.Start, ah.StartMinute),
			formatHour(ah.End, ah.EndMinute)), nil

	case 2:
		// Set/replace or clear a Per-Day Override.
		return c.setDayOverride(user, u, rest[0], rest[1])

	default:
		return "", fmt.Errorf("usage: %s", hoursUsage)
	}
}

// setDayOverride sets, replaces, or clears the Per-Day Override for the weekday
// named by dayArg. A rangeArg of "clear" removes the override; otherwise it is
// parsed as a start-end window.
func (c *AdminCommands) setDayOverride(user string, u *UserConfig, dayArg, rangeArg string) (string, error) {
	wd, err := parseWeekday(dayArg)
	if err != nil {
		return "", err
	}
	day := weekdayName(wd)

	if rangeArg == "clear" {
		if _, ok := u.AllowedHoursByDay[day]; !ok {
			return fmt.Sprintf("%s has no %s override", capitalize(user), day), nil
		}
		delete(u.AllowedHoursByDay, day)
		if err := c.cfg.save(c.savePath); err != nil {
			return "", fmt.Errorf("failed to save config: %w", err)
		}
		return fmt.Sprintf("Cleared %s override for %s", day, capitalize(user)), nil
	}

	ah, err := parseHoursRange(rangeArg)
	if err != nil {
		return "", fmt.Errorf("invalid hours: %w", err)
	}
	if u.AllowedHoursByDay == nil {
		u.AllowedHoursByDay = map[string]AllowedHours{}
	}
	u.AllowedHoursByDay[day] = ah
	if err := c.cfg.save(c.savePath); err != nil {
		return "", fmt.Errorf("failed to save config: %w", err)
	}
	return fmt.Sprintf("Updated %s allowed hours for %s: %s - %s",
		day, capitalize(user),
		formatHour(ah.Start, ah.StartMinute),
		formatHour(ah.End, ah.EndMinute)), nil
}

func (c *AdminCommands) Say(args []string) (string, error) {
	user, msg, err := c.parseUserMessage(args)
	if err != nil {
		return "", fmt.Errorf("usage: say [user] {message}: %w", err)
	}
	sendNotificationFunc(user, msg)
	sendTTSFunc(user, msg, c.cfg.TTSModel())
	return fmt.Sprintf("Sent to %s: %q", capitalize(user), msg), nil
}

func (c *AdminCommands) parseOptionalUser(args []string) (string, error) {
	switch len(args) {
	case 0:
		return c.defaultUser()
	case 1:
		return c.requireValidUser(args[0])
	default:
		return "", fmt.Errorf("too many arguments")
	}
}

func (c *AdminCommands) parseUserDuration(args []string) (string, string, error) {
	switch len(args) {
	case 1:
		if c.cfg.isValidUser(args[0]) {
			return "", "", fmt.Errorf("missing duration")
		}
		user, err := c.defaultUser()
		return user, args[0], err
	case 2:
		user, err := c.requireValidUser(args[0])
		return user, args[1], err
	default:
		return "", "", fmt.Errorf("expected duration, or user and duration")
	}
}

func (c *AdminCommands) parseUserOptionalDuration(args []string) (string, string, error) {
	switch len(args) {
	case 0:
		user, err := c.defaultUser()
		return user, "", err
	case 1:
		if c.cfg.isValidUser(args[0]) {
			return args[0], "", nil
		}
		user, err := c.defaultUser()
		return user, args[0], err
	case 2:
		user, err := c.requireValidUser(args[0])
		return user, args[1], err
	default:
		return "", "", fmt.Errorf("too many arguments")
	}
}

// splitOptionalUser resolves an optional leading user argument: if the first
// argument names a valid user it is consumed, otherwise the default user is
// resolved and all arguments are returned as the remainder.
func (c *AdminCommands) splitOptionalUser(args []string) (string, []string, error) {
	if len(args) > 0 && c.cfg.isValidUser(args[0]) {
		return args[0], args[1:], nil
	}
	user, err := c.defaultUser()
	return user, args, err
}

func (c *AdminCommands) parseUserMessage(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("missing message")
	}
	if c.cfg.isValidUser(args[0]) {
		if len(args) == 1 {
			return "", "", fmt.Errorf("missing message")
		}
		return args[0], strings.Join(args[1:], " "), nil
	}
	user, err := c.defaultUser()
	if err != nil {
		return "", "", err
	}
	return user, strings.Join(args, " "), nil
}

func (c *AdminCommands) requireValidUser(name string) (string, error) {
	if !c.cfg.isValidUser(name) {
		return "", fmt.Errorf("unknown user: %s", name)
	}
	return name, nil
}

func (c *AdminCommands) defaultUser() (string, error) {
	if len(c.cfg.Users) == 1 {
		return c.cfg.Users[0].Name, nil
	}

	var active []string
	for _, u := range c.cfg.Users {
		if getUserSessionStatusFunc(u.Name) == "active" {
			active = append(active, u.Name)
		}
	}
	switch len(active) {
	case 1:
		return active[0], nil
	case 0:
		return "", fmt.Errorf("no active configured user; specify a user")
	default:
		return "", fmt.Errorf("multiple active configured users (%s); specify a user", strings.Join(active, ", "))
	}
}
