import os

# Must be set before Qt is imported anywhere: run without a display.
os.environ.setdefault("QT_QPA_PLATFORM", "offscreen")
