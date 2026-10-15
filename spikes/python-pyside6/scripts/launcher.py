"""PyInstaller entry point (an absolute-import wrapper around lumen.app.main)."""

import sys

from lumen.app import main

if __name__ == "__main__":
    sys.exit(main())
