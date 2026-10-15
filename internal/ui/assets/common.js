// Shared by the control and output pages: a tiny API client.
// The API token (when the server requires one) arrives as ?token= on the page URL.
const token = new URLSearchParams(location.search).get("token");
const baseHeaders = token ? { "X-API-Token": token } : {};

export async function api(path, method = "GET", body) {
  const headers = body === undefined ? baseHeaders : { ...baseHeaders, "Content-Type": "application/json" };
  const res = await fetch("/api/v1" + path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  if (!res.ok) {
    const problem = await res.json().catch(() => ({}));
    throw new Error(problem.detail || problem.title || `HTTP ${res.status}`);
  }
  return res.status === 204 ? null : res.json();
}

// onState calls fn now and after every server-side state change (server-sent events).
export function onState(fn) {
  fn();
  const url = "/api/v1/events" + (token ? "?token=" + encodeURIComponent(token) : "");
  const es = new EventSource(url);
  es.addEventListener("state", fn);
  return es;
}

export const params = new URLSearchParams(location.search);
