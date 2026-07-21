# setup

## GitHub Actions

1. **Fork** this repo on GitHub.
2. **Create a Discord webhook** in the channel where Dispatch should post.
3. In your fork, open **Settings → Secrets and variables → Actions**.
4. Add a repository secret named `DISCORD_WEBHOOK_URL` containing the webhook URL.
5. Open the **Actions** tab and enable workflows if GitHub asks.
6. Run **Funkin Dispatch** manually once to test it.

The scheduled workflow checks every hour and commits the position snapshot back to `state.json`. Scheduled workflows are commonly disabled when a public repository is forked, so the fork owner must enable Actions before automatic checks begin.

The workflow needs repository write permission to save `state.json`. Do not run the GitHub workflow and a self-hosted copy against the same webhook at the same time.

## self-hosting

To run Dispatch on your own PC or server instead:

```powershell
py -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
$env:DISCORD_WEBHOOK_URL = 'https://discord.com/api/webhooks/...'
python src/run.py --loop
```

Dispatch runs immediately, then checks every hour. Use `--interval-hours` to change the interval. Keep it running with Task Scheduler on Windows or `systemd` on Linux.

## one-time testing

Use a separate state file for a test webhook:

```powershell
$env:DISCORD_WEBHOOK_URL = 'https://discord.com/api/webhooks/...'
python src/run.py --state-file .local/test-state.json --announce-existing
```

Preview the current feed without posting or changing state:

```powershell
python src/run.py --dry-run
```

## reset state

Clears the saved position snapshot so the next run starts fresh:

```powershell
powershell ./scripts/reset-state.ps1
```
