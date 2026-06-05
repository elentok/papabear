# screentimectl

Parental screen-time control daemon: enforces per-user daily time limits and the
hours of day during which a user is permitted to be logged in, controllable over Telegram.

## Language

**Allowed Hours**:
The window of the day during which a user is permitted to be logged in, as a start and end time-of-day.
_Avoid_: schedule, curfew, open hours

**Default Allowed Hours**:
The **Allowed Hours** that apply on any day without a more specific rule. This is the top-level `allowed_hours`.
_Avoid_: base hours, fallback hours

**Per-Day Override**:
An **Allowed Hours** value attached to a specific weekday that replaces the **Default Allowed Hours** for that day only. Always specifies a full start and end (never partial). Keyed by lowercase full day name (`monday`..`sunday`).
_Avoid_: exception, daily hours, custom hours

**Effective Allowed Hours**:
The **Allowed Hours** in force on a given calendar day: the **Per-Day Override** for that weekday if one exists, otherwise the **Default Allowed Hours**.
_Avoid_: resolved hours, today's hours

**Daily Limit**:
The maximum cumulative logged-in time a user may accumulate in one day, independent of **Allowed Hours**.
_Avoid_: quota, cap

## Relationships

- A **User** has exactly one set of **Default Allowed Hours** and zero or more **Per-Day Overrides**.
- A **Per-Day Override** belongs to exactly one weekday and supersedes the **Default Allowed Hours** on that day.
- The **Effective Allowed Hours** for a day is derived, never stored: override-for-that-weekday-else-default.
- **Allowed Hours** and **Daily Limit** are independent gates; a user must satisfy both to remain logged in.

## Example dialogue

> **Dev:** "If Saturday has a **Per-Day Override** of 7–18 and the **Default Allowed Hours** are 8–18, what are the **Effective Allowed Hours** on Saturday?"
> **Owner:** "7–18 — the override fully replaces the default for that day. It's not a merge; the override always carries its own start and end."
> **Dev:** "And a day with no override?"
> **Owner:** "Falls back to the **Default Allowed Hours**, 8–18."

## Flagged ambiguities

- "allowed hours" was used to mean both the single configured window and the day-specific value. Resolved: **Default Allowed Hours** (configured fallback), **Per-Day Override** (weekday-specific configured value), and **Effective Allowed Hours** (the derived value in force today) are distinct.
