"""Build a standalone Lumen bundle with PyInstaller and package it into dist/.

Usage: python scripts/build.py [--self-test]

Produces dist/lumen-<version>-<os>-<arch>.(zip|tar.gz). With --self-test the freshly built
executable is launched with ``--self-test`` to prove the bundle actually works.
"""

from __future__ import annotations

import argparse
import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DIST = ROOT / "dist"
NAME = "Lumen"


def version() -> str:
    ns: dict[str, str] = {}
    exec((ROOT / "src" / "lumen" / "__init__.py").read_text(), ns)
    return ns["__version__"]


def executable() -> Path:
    if sys.platform == "darwin":
        return DIST / f"{NAME}.app" / "Contents" / "MacOS" / NAME
    if sys.platform == "win32":
        return DIST / NAME / f"{NAME}.exe"
    return DIST / NAME / NAME


def build() -> None:
    for d in (ROOT / "build", DIST):
        shutil.rmtree(d, ignore_errors=True)
    subprocess.run(
        [
            sys.executable, "-m", "PyInstaller",
            "--noconfirm", "--clean", "--windowed",
            "--name", NAME,
            "--paths", str(ROOT / "src"),
            "--collect-submodules", "uvicorn",
            "--distpath", str(DIST),
            "--workpath", str(ROOT / "build"),
            "--specpath", str(ROOT / "build"),
            str(ROOT / "scripts" / "launcher.py"),
        ],
        check=True,
        cwd=ROOT,
    )  # fmt: skip


def self_test() -> None:
    env = dict(os.environ)
    if sys.platform.startswith("linux"):
        env.setdefault("QT_QPA_PLATFORM", "offscreen")  # no display on CI
    exe = executable()
    print(f"self-testing {exe}", flush=True)
    result = subprocess.run([str(exe), "--self-test"], env=env, timeout=120, check=False)
    if result.returncode != 0:
        sys.exit(f"self-test failed with exit code {result.returncode}")


def package() -> Path:
    arch = platform.machine().lower().replace("amd64", "x86_64").replace("aarch64", "arm64")
    os_name = {"darwin": "macos", "win32": "windows"}.get(sys.platform, "linux")
    base = DIST / f"lumen-{version()}-{os_name}-{arch}"
    if sys.platform == "darwin":
        out = base.with_suffix(".zip")
        # ditto preserves the symlinks and metadata an .app bundle needs.
        subprocess.run(
            ["ditto", "-c", "-k", "--keepParent", str(DIST / f"{NAME}.app"), str(out)], check=True
        )
    elif sys.platform == "win32":
        out = Path(shutil.make_archive(str(base), "zip", DIST, NAME))
    else:
        out = Path(shutil.make_archive(str(base), "gztar", DIST, NAME))
    return out


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--self-test", action="store_true", help="run the built app's self-test")
    args = parser.parse_args()
    build()
    if args.self_test:
        self_test()
    print(f"built {package()}")


if __name__ == "__main__":
    main()
