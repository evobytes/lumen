---
name: session-end
description: Persist everything learned in a long working session before it closes: update the decision log, session log, docs and skills in the repo, refresh the assistant's persistent memory files, and list pending decisions and next steps. Use when the user says the session is ending, or after any large chunk of work.
disable-model-invocation: true
---

# /session-end

Two stores keep this project's knowledge. Update **both**; they hold different things. The repo is the one that survives a host change: nothing important may live only in memory.

## A. The repo (shared, versioned): what is true about the project
1. Every decision made this session has a numbered entry in `docs/decisions.md` (why, evidence and its status, what it costs). New entries only; never rewrite old ones (supersede instead).
2. Append the narrative to `docs/session-log.md`: what the owner asked, what was found, what went wrong and was corrected, the state at the end. Instructions and preferences the owner stated go in its "collected" list.
3. Update `docs/roadmap.md` (proven vs unproven, open issues, next steps, first commands).
4. Run `/sync-docs` (regenerates generated docs, lints, then covers CHANGELOG, skills, AGENTS, README).
5. Anything an agent must do differently next time belongs in `AGENTS.md` (rules) or a skill (how-to), not only in memory.
6. `make docs-check` is clean.

## B. Persistent memory (private to this assistant and user): what the repo cannot say
Directory: `~/.claude/projects/-home-bevand10-git-openlp-replacement/memory/` (outside the repo: only write it as part of this skill; the project rule "write only inside the repo" does not cover the assistant's own memory). One fact per file, frontmatter `name`, `description`, `metadata.type` (`user` | `feedback` | `project` | `reference`); feedback and project bodies end with **Why:** and **How to apply:**. Index it in `MEMORY.md` (one line each, no content). Update an existing file rather than adding a duplicate; delete memories that turned out wrong; convert relative dates to absolute. Refresh at least the project status, environment facts that changed, and any new feedback or user fact. Do **not** save code structure, file lists, anything in `AGENTS.md`, skills, `docs/` or git history, or secrets.

## C. Hand-off
Report: what was persisted where; pending decisions (recommendation first); verified versus unverified; the exact first command for the next session (`make ci`, or `/verify`). Do not commit or push unless asked; the pre-commit hook (`make hooks`) will regenerate and stage docs when the user does.
