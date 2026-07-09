"""Persistent duplicate-prevention state."""

import json
from pathlib import Path

STATE_PATH = Path(__file__).resolve().parent.parent / 'state.json'


def load_state(path=STATE_PATH):
    try:
        with Path(path).open(encoding='utf-8') as state_file:
            state = json.load(state_file)
    except (FileNotFoundError, json.JSONDecodeError):
        state = {}
    state.setdefault('version', 1)
    state.setdefault('initialized', False)
    # `seen_mods` was used by the first prototype. Keep it readable so old
    # state files can be migrated without crashing, but position snapshots
    # are now the source of truth.
    state.setdefault('positions', {})
    return state


def save_state(state, path=STATE_PATH):
    state_path = Path(path)
    state_path.parent.mkdir(parents=True, exist_ok=True)
    temporary_path = state_path.with_suffix('.tmp')
    with temporary_path.open('w', encoding='utf-8') as state_file:
        json.dump(state, state_file, indent=2)
        state_file.write('\n')
    temporary_path.replace(state_path)
