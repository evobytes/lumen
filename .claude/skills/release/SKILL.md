---
name: release
description: Cut a tagged release of lumen: version bump, changelog, dry run, tag, publish verification. Use only when the user asks for a release.
---

# Release

Never start without an explicit request. Tagging and pushing are outward-facing: confirm before each.

1. Clean tree on `main`, up to date with `origin/main`.
2. Bump `VERSION` to the new `x.x.x` (semver: breaking, major; feature, minor; fix, patch). Update `CHANGELOG.md` with a `## [x.x.x]` entry and the date.
3. `make release-check TAG=vX.X.X` (tag matches `VERSION`, tree clean), then `make release-dry`: it must pass end to end. Inspect `dist/` names: `lumen_<x.x.x>_<arch>_<stamp>-<hash>.<ext>`, hash not `-dirty`.
4. **Windows check before tagging** (CI cannot prove it): on a real Windows host, `make selftest`, install the installer fresh and as an upgrade, launch, move the output to the second monitor, uninstall. If you could not, say so in the release notes.
5. Commit: `release: vX.X.X`.
6. **Ask the user**, then `git tag -a vX.X.X -m "vX.X.X"` and `git push origin main vX.X.X`.
7. Watch `release.yml`; verify the GitHub release has the zip, installer, deb, macOS zip, docs bundle, `SHA256SUMS` and attestations.

Rollback: delete the GitHub release and tag only on explicit instruction; never reuse a published version number: bump the patch.
