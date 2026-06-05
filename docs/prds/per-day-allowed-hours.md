# Per-Day Allowed Hours

## Problem Statement

As a parent administering screen time, I can only set a single **Allowed Hours**
window that applies to every day of the week. I want my child's computer to open at
7am on Saturdays and Tuesdays but stay at 8am the rest of the week, and today I can't
express that — the start and end of the allowed window are constant across all seven
days.

## Solution

Each user keeps a single **Default Allowed Hours** window plus an optional set of
**Per-Day Overrides** — a full start/end window attached to specific weekdays that
replaces the default on those days only. The daemon enforces the **Effective Allowed
Hours** for whatever weekday it currently is. I can view and edit the whole schedule
from Telegram, and the day in force is marked `(today)` wherever the schedule is shown.

## User Stories

1. As a parent, I want to keep configuring a single allowed-hours window that applies
   to every day, so that my existing setup keeps working unchanged.
2. As a parent, I want to add a per-day override for Saturday, so that the computer
   opens earlier on weekends.
3. As a parent, I want to add a per-day override for Tuesday, so that an individual
   weekday can differ from the rest without affecting other weekdays.
4. As a parent, I want each override to carry its own full start and end, so that the
   resulting window for that day is unambiguous.
5. As a parent, I want days without an override to fall back to the default window, so
   that I only have to specify the days that differ.
6. As a parent, I want the daemon to enforce the override on the matching weekday and
   the default on all other days, so that lock/unlock behavior follows the schedule.
7. As a parent, I want an invalid day name in the config (e.g. `saterday`) to be a hard
   startup error, so that a typo never silently goes unenforced.
8. As a parent, I want an override whose start is not before its end to be rejected, so
   that I can't accidentally create a window that's always closed.
9. As a parent, I want to run `/hours bob` and see the full schedule — default plus
   every override — so that I can review the whole week at a glance.
10. As a parent, I want the line that applies to the current weekday marked `(today)`,
    so that I can immediately see what's in force right now.
11. As a parent, I want `/hours bob 8-18` to set the default window, so that the
    existing no-day command still works.
12. As a parent, I want `/hours bob saturday 7-18` to set or replace Saturday's
    override, so that I can change a single day from my phone.
13. As a parent, I want `/hours bob saturday clear` to remove Saturday's override, so
    that Saturday falls back to the default again.
14. As a parent, I want `/status bob` to show the full schedule with `(today)` marked,
    so that a status check also tells me the active window.
15. As a parent, I want a clear error message when I pass an unknown day or a bad range
    to `/hours`, so that I can correct the command.
16. As a parent, I want the tray's status endpoint to report today's effective window,
    so that any current/future tray display reflects the day-specific hours.
17. As a parent, I want the example config to document the per-day override syntax, so
    that I know how to write it by hand.

## Implementation Decisions

### Schema (Config / `UserConfig`)

- Add `AllowedHoursByDay map[string]AllowedHours` to `UserConfig`, YAML key
  `allowed_hours_by_day`. The existing `allowed_hours` field is unchanged and serves as
  the **Default Allowed Hours**.
- Backward compatible: a config with no `allowed_hours_by_day` behaves exactly as today.
- Day keys are **lowercase full names** (`monday`..`sunday`). Validation at load time:
  - Unknown day key → hard load error.
  - Each override must satisfy `start*60+startMinute < end*60+endMinute` → else hard
    load error (same rule as the `/hours` range parser).
- The existing 8–18 default fallback in `loadConfig` applies **only** to the top-level
  `allowed_hours` (when both Start and End are 0). It is never applied to overrides, so
  an override of `{start: 0, end: 8}` (midnight–8am) is honored literally.
- Overrides must specify a full start and end; partial overrides (inherit one bound from
  the default) are not supported.

### Effective-hours resolver (deep, pure module)

- A pure function resolves the **Effective Allowed Hours** for a given weekday:
  override-for-that-weekday-if-present, else the **Default Allowed Hours**. No clock, no
  I/O — the weekday is passed in.
- Approximate shape:
  ```go
  func effectiveAllowedHours(def AllowedHours, byDay map[string]AllowedHours, wd time.Weekday) AllowedHours
  ```
- `isWithinAllowedHours` and all its current call sites (`session.go`, `main.go`) consume
  the resolved value for `time.Now()`'s weekday. The clock read stays at the call site
  (or a thin wrapper); the resolver itself takes the weekday explicitly so it is testable
  without faking time.

### Day-name parser (small pure helper)

- Maps a lowercase day word to `time.Weekday`, returning an error for unknown words.
  Used by both config load validation and the `/hours` command.

### Schedule formatter (deep, pure module)

- A pure function renders the full schedule as display text: the default line plus one
  line per override, with `(today)` appended to whichever line is in force for the passed
  weekday. Given default + overrides + today's weekday → string (or lines). Shared by
  `/status` and the `/hours` read path so they cannot drift.

### `/hours` Telegram command (`commands.go`, `telegram.go`)

Argument grammar (user is resolved as today):

- `/hours [user]` → show full schedule via the formatter (with `(today)`).
- `/hours [user] start-end` → set the **Default Allowed Hours**.
- `/hours [user] <day> start-end` → set/replace that weekday's **Per-Day Override**.
- `/hours [user] <day> clear` → remove that weekday's override (falls back to default).

After any mutation the config is saved (existing atomic `save`) and a confirmation is
returned. Unknown day or invalid range → descriptive error. The existing `parseHoursRange`
/ `parseHour` parsers are reused unchanged for the range token.

### HTTP status endpoint (`http.go`)

- Keep the four existing fields (`allowed_hours_start`, `allowed_hours_start_minute`,
  `allowed_hours_end`, `allowed_hours_end_minute`) — JSON contract unchanged — but
  populate them with **today's Effective Allowed Hours** rather than the raw default.

### Example config (`assets/example-config.yaml`)

- Add a commented `allowed_hours_by_day` example showing two weekday overrides.

## Testing Decisions

Good tests here assert externally observable behavior — the resolved window, the rendered
schedule text, accept/reject of a config, command output and config mutation — not
internal structure. Prior art: `config_test.go`'s `loadTestConfig` table style and
`session_test.go`'s function-stub pattern (`isWithinAllowedHoursFunc`).

Modules to test:

1. **Effective-hours resolver** — table tests over all weekdays: no overrides → default
   every day; override on one day → that day differs, others stay default; multiple
   overrides; midnight-boundary override honored literally.
2. **Schedule formatter** — table tests: default only; default + overrides; `(today)`
   placed on the correct line for various weekdays (including a day with no override →
   marker on the default line).
3. **Config load/validate** (extends `config_test.go`) — valid override parses; unknown
   day key errors; override with start ≥ end errors; default fallback applies only when
   no default and no overrides; presence of overrides does not trigger the default
   fallback.
4. **`/hours` command** — set default; set a day override; replace an existing override;
   `clear` an override; `clear` a non-existent override; show schedule; unknown day
   error; bad range error. Reuse the existing `AdminCommands` test setup.

## Out of Scope

- Per-day **Daily Limit** (only **Allowed Hours** become per-day; the daily time cap
  stays global per user).
- Partial overrides that inherit one bound from the default.
- Date-specific overrides (holidays, one-off dates) — only weekday recurrence.
- Case-insensitive or abbreviated day names — lowercase full names only.
- Timezone configuration — resolution uses the machine's local time, as today.
- Tray UI changes — the Python tray does not currently read the allowed-hours fields;
  only the JSON it could consume is kept correct.

## Further Notes

- Domain vocabulary is defined in `CONTEXT.md`: **Default Allowed Hours**, **Per-Day
  Override**, **Effective Allowed Hours**.
- No ADR: the schema is backward compatible, unsurprising, and the trade-off (vs. a full
  7-day map) is minor.
