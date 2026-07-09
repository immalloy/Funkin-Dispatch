"""Discord webhook publishing."""

import json
import os
import urllib.request
from datetime import datetime, timezone

from gamebanana import period_label


def _webhook_base():
    url = os.environ.get('DISCORD_WEBHOOK_URL', '').strip().rstrip('/')
    if not url:
        raise RuntimeError('DISCORD_WEBHOOK_URL is not set')
    parts = url.split('/')
    if len(parts) < 2 or not parts[-1] or not parts[-2]:
        raise RuntimeError('DISCORD_WEBHOOK_URL does not look valid')
    return f'https://discord.com/api/webhooks/{parts[-2]}/{parts[-1]}'


def post_mod(period, rank, mod, config, previous=None):
    submitter = mod.get('_aSubmitter') or {}
    name = mod.get('_sName') or 'Untitled mod'
    mod_url = mod.get('_sProfileUrl') or ''
    author = submitter.get('_sName') or 'Unknown creator'
    author_url = submitter.get('_sProfileUrl') or ''
    image_url = mod.get('_sImageUrl')

    if previous:
        period_changed = previous['period'] != period
        title = 'Featured Period Change' if period_changed else 'Featured Ranking Update'
        movement = (f'{period_label(previous["period"])} #{previous["rank"]} '
                    f'-> {period_label(period)} #{rank}')
        description = f'Moved from **{movement}**.'
    else:
        title = 'New Mod Featured'
        description = f'Now featured in **{period_label(period)} #{rank}**.'

    embed = {
        'author': {'name': author, 'url': author_url},
        'title': name,
        'url': mod_url,
        'description': description,
        'color': 0xe74c3c,
        'footer': {'text': 'Funkin Dispatch'},
        'timestamp': datetime.now(timezone.utc).isoformat(),
    }
    if image_url:
        if previous and previous['period'] == period:
            embed['thumbnail'] = {'url': image_url}
        else:
            embed['image'] = {'url': image_url}
    if submitter.get('_sAvatarUrl'):
        embed['author']['icon_url'] = submitter['_sAvatarUrl']

    payload = {
        'username': config.get('webhook_username', 'Funkin Dispatch'),
        'embeds': [embed],
        'allowed_mentions': {'parse': []},
    }
    request = urllib.request.Request(
        f'{_webhook_base()}?wait=true',
        data=json.dumps(payload).encode('utf-8'),
        headers={'Content-Type': 'application/json', 'User-Agent': 'Funkin-Dispatch/1.0'},
        method='POST',
    )
    with urllib.request.urlopen(request, timeout=20) as response:
        return json.loads(response.read())
