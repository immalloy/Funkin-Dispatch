# config

Edit `config.json` in the repository root.

## periods

Periods are checked in the order listed. If a mod appears in more than one period, the first matching period is used.

```json
"periods": ["today", "week", "month", "3month", "6month", "year", "alltime"]
```

Available values:

- `today`
- `week`
- `month`
- `3month`
- `6month`
- `year`
- `alltime`

## max per period

```json
"max_per_period": 3
```

How many entries to use from each GameBanana period. The API currently returns
three entries per period; keeping this configurable allows that limit to
change without changing the code.

## announcement types

```json
"announce_new_mods": true,
"announce_rank_changes": true,
"announce_period_changes": true
```

- `announce_new_mods` posts when a mod first enters the current feed.
- `announce_rank_changes` posts when a mod moves within the same period.
- `announce_period_changes` posts when a mod moves between periods.

## first run

```json
"announce_existing_on_first_run": false
```

When `false`, the first run records current positions without posting them. Set it to `true` if the first run should announce the existing feed.

## interval

```json
"interval_hours": 1
```

The delay between checks when using `--loop`. The command-line option `--interval-hours` overrides this value.

## filters

```json
"blacklist": ["nsfw", "18+", "gore"],
"show_flagged_content": false
```

Blacklist terms are matched against mod names and creator names. Flagged content is excluded by default.
