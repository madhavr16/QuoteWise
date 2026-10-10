"""Pytest configuration: make the ``app`` package importable from the
service root regardless of the directory tests are invoked from."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
