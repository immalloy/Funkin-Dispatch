"""Entry point for Funkin Dispatch."""

import argparse
import time
from datetime import datetime, timezone

from config import load_config
from discord_webhook import post_mod
from gamebanana import fetch_featured_mods
from state import load_state, save_state


def parse_args():
    parser = argparse.ArgumentParser(description='Announce newly featured FNF mods.')
    parser.add_argument('--loop', action='store_true', help='Keep checking at the configured interval.')
    parser.add_argument('--interval-hours', type=float, help='Override config.json interval.')
    parser.add_argument('--state-file', help='Use a separate state file for testing.')
    parser.add_argument('--dry-run', action='store_true', help='List new mods without posting or marking them seen.')
    parser.add_argument('--announce-existing', action='store_true', help='Announce the current feed on this run, including first-run entries.')
    return parser.parse_args()


def run_once(args, config):
    state = load_state(args.state_file) if args.state_file else load_state()
    candidates = fetch_featured_mods(config)
    print(f'[ok] fetched {len(candidates)} featured mod(s) across {len(config.get("periods", []))} period(s)')
    previous_positions = state['positions']
    current_positions = {
        str(mod['_idRow']): {'period': period, 'rank': rank}
        for period, rank, mod in candidates
    }
    changes = [
        (period, rank, mod, previous_positions.get(str(mod['_idRow'])))
        for period, rank, mod in candidates
        if previous_positions.get(str(mod['_idRow'])) != {'period': period, 'rank': rank}
    ]

    if args.dry_run:
        for period, rank, mod, previous in changes:
            label = 'update' if previous else 'new'
            print(f'[{label}] {mod.get("_sName", "Untitled mod")}: '
                  f'{previous["period"]} #{previous["rank"]} -> {period} #{rank}'
                  if previous else f'[{label}] {mod.get("_sName", "Untitled mod")}: {period} #{rank}')
        print(f'[done] found {len(changes)} position change(s)')
        return

    if (not state['initialized'] and
            not (args.announce_existing or config.get('announce_existing_on_first_run', False))):
        state['positions'] = current_positions
        state['initialized'] = True
        if not args.dry_run:
            save_state(state, args.state_file) if args.state_file else save_state(state)
        print(f'[ok] initialized with {len(candidates)} current position(s); nothing posted')
        return

    next_positions = {}
    posted = 0
    skipped = 0
    # Post the longest period first and the daily period last. Reverse the
    # ranking order within each period so the lowest rank is posted first.
    posting_order = {
        period: index for index, period in enumerate(config.get('periods', []))
    }
    posting_candidates = sorted(
        candidates,
        key=lambda candidate: (posting_order.get(candidate[0], -1), candidate[1]),
        reverse=True,
    )
    for period, rank, mod in posting_candidates:
        mod_id = str(mod['_idRow'])
        previous = previous_positions.get(mod_id)
        if not previous:
            # New entries are announced below. Existing entries that came from
            # the old seen_mods format are adopted without a duplicate post.
            old_seen = state.get('seen_mods', {}).get(mod_id)
            if old_seen:
                next_positions[mod_id] = {'period': period, 'rank': rank}
                continue
        if previous == {'period': period, 'rank': rank}:
            next_positions[mod_id] = previous
            continue
        is_new = previous is None
        period_changed = previous is not None and previous['period'] != period
        rank_changed = previous is not None and previous['period'] == period
        if ((is_new and not config.get('announce_new_mods', True)) or
                (period_changed and not config.get('announce_period_changes', True)) or
                (rank_changed and not config.get('announce_rank_changes', True))):
            next_positions[mod_id] = {'period': period, 'rank': rank}
            print(f'[skip] {mod.get("_sName", "Untitled mod")} change disabled by config')
            skipped += 1
            continue
        try:
            post_mod(period, rank, mod, config, previous)
            next_positions[mod_id] = {'period': period, 'rank': rank}
            print(f'[ok] announced {mod.get("_sName", "Untitled mod")}')
            posted += 1
        except Exception as exc:
            print(f'[warn] failed to announce {mod.get("_sName", "Untitled mod")}: {exc}')

    state['positions'] = next_positions
    state['initialized'] = True
    if next_positions != previous_positions:
        save_state(state, args.state_file) if args.state_file else save_state(state)
        print(f'[done] checked {len(candidates)} featured mod(s); posted {posted}, skipped {skipped} (state updated)')
    else:
        print(f'[done] checked {len(candidates)} featured mod(s); posted {posted}, skipped {skipped} (no changes)')


def main():
    args = parse_args()
    config = load_config()
    interval = args.interval_hours or config.get('interval_hours', 1)
    if interval <= 0:
        raise ValueError('interval must be greater than zero')
    if not args.loop:
        run_once(args, config)
        return
    print(f'[ok] dispatch loop started; checking every {interval:g} hour(s)')
    while True:
        try:
            run_once(args, config)
        except Exception as exc:
            print(f'[fatal] check failed: {exc}')
        time.sleep(interval * 60 * 60)


if __name__ == '__main__':
    main()
