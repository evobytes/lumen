# Reference projects: what was reviewed, ranked, and adopted

Reviewed 2026-10-14 to decide the build, CI, docs and agent conventions (decision 013). The owner's instruction: find peers such as
`tips-go` and `hive-nmos-capture`, and **judge them by age and content, not by name**. All were read read-only.

## Ranking (by recency and relevance)

| Repo | Where | Last activity | Relevance | Verdict |
|---|---|---|---|---|
| `hive-nmos-capture` | GitHub `dB-Broadcast` (private), not cloned locally | pushed 2026-10-13; 19 skills, 44 decisions, docgen | **Windows-first Go app**: Windows is the primary CI job, a Linux job adds `-race`, `make vet`/`lint` for both OSes, Inno Setup installer, script-mode check, `windows-service` skill, `.vscode` Git Bash settings | **Primary model** |
| `tips-go` | `~/git/tips-go`, GitHub `evobytes` | files 2026-10-11; 22 skills; no commits | newer than its template (adds an intent document, a data-model document and a MOS skill) but **Linux-only** (deb, rpm, systemd), htmx/Bootstrap UI, SQLite/Postgres | Secondary: Makefile, packaging with nfpm, pre-commit hook, release workflow, ADR style |
| `reference-go-app-template` | `~/git/reference-go-app-template` | files 2026-10-10; 21 skills | **A stale snapshot of `tips-go`** (its remote is `evobytes/tips-go`), despite the name | Lowest authority: read for the baseline only |
| `hive-cli-tools` | `~/git/hive-cli-tools`, GitHub `dB-Broadcast` | 2026-10-08; 3 skills | release automation scripts (`compute-next-version`, `generate-release-notes`, `publish-release`, `smoke-test-package`, `ci-summary`) | Source of the `scripts/build-*` naming and future release-automation ideas |
| `hive-driver-evs-cerebrum`, `hive-www-apps` | `~/git` | 2026-10 | `scripts/build-deb` is a thin wrapper that execs `scripts/build-package deb` (fpm) | Source of the `build-deb` name |
| `hive-repo-server`, `hive-netbox-deployer` | `~/git` | 2026-10 | apt/rpm repository server and deb/rpm build scripts | Not used; relevant if Lumen ever needs an apt repository |
| `tips`, `tips2026`, `tips-running-orders` | `~/git` | 2024-2026-07 | older, no AGENTS/docgen | Not used |

## Adopted (and from where)
- **`make` as the only interface**, logic in `scripts/`, CI calls only `make` (all).
- **Windows as the primary CI job, Linux adds the race detector, both run the same `make ci CHECK_OS=host`** (`hive-nmos-capture`). Adapted: Linux and macOS cannot be cross-vetted (CGO), so `CHECK_OS` is `windows` plus the host (decision 017).
- **`make vet`/`lint` for more than the host OS** and the script-mode check `exec-check` (`hive-nmos-capture`).
- **Inno Setup installer** driven by a script that passes version and copyright from `COPYRIGHT` (`hive-nmos-capture`).
- **`VERSION` plus `<stamp>-<hash>` artifact names**, `scripts/version`, `release-check` (all).
- **`os/` mirrors the target filesystem**, nfpm for the deb with explicit file lists (`tips-go`).
- **docgen** (generate, check, lint, staged), the pre-commit hook, generated `NOTICE`/`THIRD_PARTY_NOTICES.md`/Debian copyright (`hive-nmos-capture`, `tips-go`); extended with an API reference and a reproducible docs bundle.
- **Pinned tools in `.tools/bin`** and SHA-pinned actions with attestation and checksums on release (all).
- **`AGENTS.md` with a `CLAUDE.md` symlink, skills, numbered ADRs, `session-end`** (all). Extended with `docs/intent.md` (`tips-go`), `docs/roadmap.md` and `docs/session-log.md` (new).
- **Apache-2.0 and a permissive-only dependency gate** (all; copyright holder follows `tips-go`).

## Deliberately not adopted
- Windows service, SCM, Event Log, Npcap handling (`hive-nmos-capture`): Lumen is a desktop app, not a service.
- htmx, Bootstrap, SQLite/PostgreSQL, REMOTE_USER proxy auth, `e2e` puppeteer tests (`tips-go`): Lumen's pages are plain modules with a shared-token API; a browser test suite may come later.
- `CGO_ENABLED=0` everywhere: not possible for Wails on Linux/macOS (decision 017).
- rpm packaging and arm64 Windows: not requested (arm64 Linux deb is built in CI).
- Developing natively on Windows with Git Bash (`hive-nmos-capture` decision 024): Lumen starts in WSL2 with interop (decision 011), with the native route documented.

## Caveats
- The SHA pins for GitHub Actions were copied from these repositories and have not been re-verified.
- `hive-nmos-capture` was read through the GitHub API into a git-ignored `.ref/` directory (deleted at the end of the session); `tips-go` and the template were read in place.
