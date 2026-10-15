---
name: output-page
description: Conventions for the control and output web pages in lumen: output as a pure function of state, event-driven refresh, strict CSP, no build step, the probe and debug modes. Use when editing anything under internal/ui.
---

# The pages (`internal/ui/assets`)

## Principle
The output page is a **pure function of server state**: on load and on every `state` server-sent event it fetches `/api/v1/status` and re-renders. It owns no state of its own, so a Wails window, OBS, CasparCG, a kiosk browser or a second PC can all render the same URL and stay in step (decision 004). The control page is the same idea plus buttons that call the API.

## Files
`output.html`/`output.js`/`output.css`, `control.html`/`control.js`, `app.css`, `common.js` (the API client and `onState`). Native ES modules loaded with `<script type="module" src="/assets/...">`. Embedded by `internal/ui` (restart to see changes).

## Rules
- **No build step, no framework, no npm.** Plain modern JS and CSS.
- **Strict CSP**: no inline `<script>`, no `style=""`, no `onclick=`, no `eval`; styles and scripts are files. `internal/api` sends `default-src 'self'` and a test fails if a page contains inline script or style.
- **Slide text is text**: set `textContent`, never `innerHTML`.
- Sizes in `vmin`/`vw`, not pixels: the page must not depend on `devicePixelRatio` (it disagreed with Win32 on the first host).
- Hide the cursor on the output; black background; blank is a CSS class that hides the slide, not a different page.
- The token (when configured) arrives as `?token=` on the page URL; `common.js` forwards it as `X-API-Token`, and on the event stream as `?token=`.

## Modes (query parameters on `/output`)
- `?debug`: blue background and an overlay with the viewport size, `devicePixelRatio`, visibility and a clock, and posts `render` reports. Use it with a screenshot so a blank monitor is distinguishable from a not-yet-loaded page.
- `?probe`: the media probe used by `lumen -selftest`: `canPlayType` matrix, real playback of the clips served at `/media/`, VP9 alpha pixel check, `setSinkId` presence, audio output device count; posts one `media` report.

## Adding a field the audience sees
Add it to `state.Snapshot`, render it in `output.js`, update `control.js` if the operator needs it, keep the strict CSP, and test the snapshot in `internal/state` and the route in `internal/api`.
