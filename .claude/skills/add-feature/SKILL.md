---
name: add-feature
description: End-to-end checklist for adding a feature (API route, state change, window behavior, output-page behavior, config key) to lumen. Use first for any non-trivial change.
---

# Add a feature

Work inside-out and keep `make ci` green at each step. Check `docs/intent.md` first: the feature should trace to it (or change it in the same commit), and `docs/roadmap.md` for where it sits.

1. **Pure logic first**: put the decision in a portable package with a table-driven test (`internal/state` is the model). Window, COM or process work comes last and stays thin.
2. **Interface at the consumer**: if it needs the OS or the windows, add to `internal/host` (small interface, physical pixels) and implement it in `internal/host/wailshost`, with a fake in `internal/host/hosttest`. Never import Wails elsewhere.
3. **Route**: add it to the route table in `internal/api/api.go` (method, path, summary, handler). `docs/api.md` regenerates from that table. Errors are problem+json; bodies are bounded strict JSON; anything derived from a request is validated (a monitor ID must be one the OS reported; no request names a path, URL or executable).
4. **Both renderers**: the output page and the control page re-read `/api/v1/status` on every event. A new field in the state snapshot needs both pages (and, if it affects the audience, the output) updated in `internal/ui/assets`; keep the CSP (no inline script or style).
5. **Config key**: add the field with `toml` and `doc` tags in `internal/config/config.go`, an `envKeys` entry and validation; `make docs` regenerates the reference and example files.
6. **Tests**: handler tests with `httptest` including the auth cases; state tests; for window behavior add a self-test check (`internal/selftest`) so it can be verified on a real machine.
7. **Security pass**: `security` skill.
8. **Docs**: `CHANGELOG.md`, `docs/intent.md` if the intent changed, the relevant skill, and a numbered entry in `docs/decisions.md` if you decided something. `make docs`.
9. **Verify**: `/verify`; say what is only compile-checked for Windows.

Keep each change small and reviewable; do not mix refactors with features.
