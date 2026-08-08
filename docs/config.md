# Configuration

Edit `config.json` in the repository root.

## Periods

Periods are checked in the order listed. If a mod appears in more than one period, the first matching period is used.

```json
"periods": ["today", "week", "month", "3month", "6month", "year", "alltime"]
```

Available values are `today`, `week`, `month`, `3month`, `6month`, `year`, and `alltime`.

## How many mods to use

```json
"max_per_period": 3
```

This is the number of entries taken from each GameBanana period.

## Announcement types

```json
"announce_new_mods": true,
"announce_rank_changes": true,
"announce_period_changes": true
```

- `announce_new_mods` posts when a mod first enters the current feed.
- `announce_rank_changes` posts when a mod moves within the same period.
- `announce_period_changes` posts when a mod moves between periods.

Dispatch also posts when a tracked mod leaves the current feed.

## First run

```json
"announce_existing_on_first_run": false
```

When `false`, the first run records the current positions without posting them. Set it to `true` when the first run should announce the existing feed.

## Interval

```json
"interval_hours": 1
```

The delay between checks when using `--loop`. The `--interval-hours` command-line option overrides it for that run.

## Filters

```json
"blacklist": ["nsfw", "18+", "gore"],
"show_flagged_content": false
```

Blacklist terms are matched against mod names and creator names. Flagged content is excluded by default.

## Embed colors

The Go runtime keeps event colors in one mapping:

- new mod: green (`#57F287`)
- ranking change: blue (`#5865F2`)
- period change: purple (`#9B59B6`)
- no longer featured: red (`#ED4245`)
