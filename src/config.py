"""Configuration loading for Funkin Dispatch."""

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CONFIG_PATH = ROOT / 'config.json'


def load_config(path=CONFIG_PATH):
    with Path(path).open(encoding='utf-8') as config_file:
        config = json.load(config_file)

    periods = config.get('periods', ['today'])
    if not periods:
        raise ValueError('config.json must enable at least one period')
    return config
