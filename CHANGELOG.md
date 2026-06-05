# Changelog

## v0.5.0 - 2026-06-05

- Renamed the project to Papa Bear.
- Reworked the CLI around Cobra, which improves built-in help output and adds shell completion generation for bash, fish, zsh, and PowerShell.
- Added a `config` command with `config show` and `config edit` subcommands for inspecting the compiled config and editing it with validation.
- Added per-day allowed-hours overrides, including `hours [user] <day> start-end` and `hours [user] <day> clear`, and updated status output to show the full weekly schedule with overrides.
- Status output now includes user account expiration state, and overnight unlock handling was fixed so allowed users are unlocked correctly after the machine is left running overnight or restarted.
- The daemon now reloads config and usage data during polling so CLI-admin changes take effect promptly, and shutdown unlocks all managed users cleanly.
- TTS delivery is faster because speech is triggered after sending the desktop notification, and the piper installer/setup flow was fixed and refreshed.
- Deploy/setup behavior was updated to remove old `screentimectl` installs and refresh related sudoers/runtime assets.

## v0.4.0

- Switched account lock/unlock from `passwd -l` / `passwd -u` to `chage -E 0` / `chage -E -1` to avoid unlock failures on accounts without a usable password hash.
- Added `papabear status --compact` and a GNOME AppIndicator tray helper installed by `setup` to show remaining screen time from the user's session.
- Telegram commands can now omit the user argument when there is one configured user or exactly one configured active user.
- Added SSH-friendly CLI equivalents for Telegram admin actions: `give`, `lock`, `unlock`, `status [user]`, `hours`, and `say` now share the same parsing and default-user resolution.
- Added `unlock [user] {duration}` / `/unlock [user] {duration}` to set remaining time and allow login. Positive `lock [user] [duration]` still works as a compatibility alias.
- `setup` now installs all runtime apt dependencies, including notification and TTS packages.
- Replaced `espeak-ng` with `piper-tts` for higher-quality TTS. `setup` installs piper-tts into a system-wide venv (`/usr/local/lib/piper-tts/`) and downloads the `en_US-lessac-medium` voice model. TTS output is cached so repeated messages (warnings, expiry notices) are played instantly without re-running inference.
- `tts` config simplified to a single `model` field (e.g. `en_US-lessac-medium`); `voice` and `fallback_voices` are removed.
- Fixed: if the computer is shut down while a user account is locked (time expired), the account is now automatically unlocked on daemon startup when the user is within their allowed hours and has time remaining.

## v0.3.0

Activity logging and status timeline.

### What changed

**Activity log** -- The daemon now tracks status transitions (active, locked, idle, offline) and writes them to per-user daily JSONL files at `/var/lib/papabear/log/{user}/YYYY-MM-DD.log`. Only transitions are logged, not every poll tick.

**Timeline in /status** -- The `/status` command (Telegram, CLI, and HTTP) now shows a timeline of the day's activity:

```
Today:
  08:00-10:30 (2h 30m) - active
  10:30-11:00 (30m) - locked
  11:00-14:30 (3h 30m) - active
```

**Shutdown logging** -- When the daemon receives SIGTERM (systemd stop or system poweroff), a `shutdown` entry is written to the activity log.

**Time grant notifications** -- When a parent grants more time via `/give`, the child now receives a desktop notification and TTS announcement with the updated remaining time.

### Upgrade notes

1. Run `sudo papabear setup` to create the new `/var/lib/papabear/log/` directory
2. Restart the service: `sudo systemctl restart papabear`

## v0.2.0

Replace timekpr-next with standalone session management.

### Why

timekpr-next continued counting screen time while the machine was locked, causing the child to lose time while away from the computer. This release removes the timekpr-next dependency entirely and manages screen time directly.

### What changed

**Session tracking** -- The daemon now polls `loginctl` every 10 seconds to detect active sessions. Time only counts when the session is active (not locked, not idle).

**Account locking** -- When time expires or the child is outside allowed hours, the screen is locked via `loginctl lock-session` and the account is locked via `passwd -l` to prevent re-login. Accounts are unlocked on day reset or when parents grant time.

**PAM integration** -- A new `check-login` command is installed as a PAM rule to prevent login when outside allowed hours or when no time remains. Parents can override this via `/give`.

**Notifications and TTS** -- Desktop notifications (`notify-send`) and spoken alerts (`espeak-ng`) fire at configurable thresholds (default: 30, 15, 5, 1 minutes remaining).

**New Telegram commands:**

- `/hours bob 8-20` -- view or set allowed login hours
- `/say bob message` -- speak a message to the child via TTS

**New CLI commands:**

- `papabear status` -- for the child to check their remaining time
- `papabear ask` -- for the child to request more time
- `papabear hours bob 8-20` -- view or set allowed hours from SSH
- `papabear say bob message` -- speak a message via TTS
- `papabear check-login` -- PAM login check

**Configuration:**

- Added `daily_limit_minutes` and `allowed_hours` per user
- Added `notifications.thresholds` for alert timing
- Usage data stored in `/var/lib/papabear/usage.json`

### Removed

- timekpr-next dependency (no longer needed)

### Upgrade notes

1. Install `espeak-ng` and `libnotify-bin` if not already present
2. Run `sudo papabear setup` to install new sudoers rules, PAM rule, and data directory
3. Update `/etc/papabear/config.yaml` to add `daily_limit_minutes` and `allowed_hours` per user (defaults to 300 minutes and 8am-6pm if omitted)
4. Restart the service: `sudo systemctl restart papabear`
5. timekpr-next can be uninstalled if no longer needed
