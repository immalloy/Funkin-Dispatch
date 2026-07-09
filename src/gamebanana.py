"""GameBanana featured-mod source."""

import json
import urllib.request

from config import load_config

USER_AGENT = 'Funkin-Dispatch/1.0'

PERIOD_LABELS = {
    'today': 'Today', 'week': 'This Week', 'month': 'This Month',
    '3month': '3 Months', '6month': '6 Months', 'year': 'This Year',
    'alltime': 'All Time',
}


def period_label(period):
    return PERIOD_LABELS.get(period, period)


def fetch_featured_mods(config):
    game_id = config.get('game_id', 8694)
    url = f'https://gamebanana.com/apiv12/Game/{game_id}/TopSubs'
    request = urllib.request.Request(url, headers={'User-Agent': USER_AGENT})
    with urllib.request.urlopen(request, timeout=20) as response:
        items = json.loads(response.read())

    periods = config.get('periods', ['today'])
    blacklist = [term.casefold() for term in config.get('blacklist', [])]
    show_flagged = config.get('show_flagged_content', False)
    results = []
    seen = set()

    for period in periods:
        rank = 0
        for mod in items:
            if mod.get('_sPeriod') != period:
                continue
            if not show_flagged and mod.get('_sInitialVisibility') != 'show':
                continue
            name = (mod.get('_sName') or '').casefold()
            author = (mod.get('_aSubmitter', {}).get('_sName') or '').casefold()
            if any(term in name or term in author for term in blacklist):
                continue
            rank += 1
            mod_id = str(mod.get('_idRow', ''))
            if not mod_id or mod_id in seen:
                continue
            seen.add(mod_id)
            results.append((period, rank, mod))

    return results
