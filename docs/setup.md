# Setup

Dispatch is a small Go program. You need Go installed and a Discord webhook for the channel where it should post.

## GitHub Actions

1. Fork this repo on GitHub.
2. Create a Discord webhook in the channel where Dispatch should post.
3. In your fork, open **Settings → Secrets and variables → Actions**.
4. Add a repository secret named `DISCORD_WEBHOOK_URL` with the webhook URL.
5. Enable Actions if GitHub asks you to.
6. Run **Funkin Dispatch** manually once to test it.

The scheduled workflow checks every hour and commits the position snapshot back to `state.json`. Scheduled workflows are commonly disabled on forked public repositories, so the fork owner needs to enable Actions first.

The workflow needs repository write permission to save `state.json`. Do not run the workflow and a self-hosted copy against the same webhook at the same time.

## Run it yourself

From the repository root:

```powershell
$env:DISCORD_WEBHOOK_URL = 'https://discord.com/api/webhooks/...'
go run . --loop
```

Dispatch runs immediately, then checks again at the configured interval. Use `--interval-hours` to override the interval for one run. You can keep it running with Task Scheduler on Windows or `systemd` on Linux.

To build a standalone executable:

```powershell
go build -o funkin-dispatch.exe .
.\funkin-dispatch.exe --loop
```

## Test without posting

Use a separate state file when testing a real webhook:

```powershell
$env:DISCORD_WEBHOOK_URL = 'https://discord.com/api/webhooks/...'
go run . --state-file .local/test-state.json --announce-existing
```

To preview changes without posting or changing state:

```powershell
go run . --dry-run
```

## Reset state

This clears the saved position snapshot so the next run starts fresh:

```powershell
powershell ./scripts/reset-state.ps1
```
