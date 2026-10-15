"""A tiny phone-friendly remote control page, served at ``/``."""

REMOTE_HTML = """<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Lumen remote</title>
<style>
  :root { color-scheme: light dark; --bg:#f4f4f6; --fg:#16161a; --card:#fff; --accent:#3b6cf6; }
  @media (prefers-color-scheme: dark) { :root { --bg:#16161a; --fg:#eee; --card:#24242b; } }
  * { box-sizing: border-box; }
  body { margin:0; padding:16px; font:16px system-ui,sans-serif; background:var(--bg); color:var(--fg); }
  main { max-width:640px; margin:0 auto; display:grid; gap:12px; }
  .card { background:var(--card); border-radius:12px; padding:16px; }
  #current { white-space:pre-wrap; font-size:1.25rem; min-height:5em; }
  .row { display:grid; grid-template-columns:1fr 1fr 1fr; gap:12px; }
  button { font:inherit; padding:20px 8px; border:0; border-radius:12px; background:var(--accent); color:#fff; }
  button.alt { background:#6b7280; } button.on { background:#c0392b; }
  ol { margin:0; padding:0; list-style:none; display:grid; gap:4px; }
  li button { width:100%; text-align:left; padding:10px 12px; background:transparent; color:inherit; }
  li.active button { background:var(--accent); color:#fff; }
  small { opacity:.7; }
</style>
</head>
<body>
<main>
  <div class="card"><small id="pos"></small><div id="current">Connecting…</div></div>
  <div class="row">
    <button id="prev">◀ Prev</button>
    <button id="blank" class="alt">Blank</button>
    <button id="next">Next ▶</button>
  </div>
  <div class="card"><ol id="slides"></ol></div>
</main>
<script>
const token = new URLSearchParams(location.search).get("token");
const headers = token ? { "X-API-Token": token } : {};
const api = (path, method = "GET", body) =>
  fetch("/api/v1" + path, {
    method,
    headers: body ? { ...headers, "Content-Type": "application/json" } : headers,
    body: body ? JSON.stringify(body) : undefined,
  }).then(r => r.ok ? r.json() : Promise.reject(r.status));

let listKey = "";
async function refresh() {
  try {
    const [st, slides] = await Promise.all([api("/status"), api("/slides")]);
    document.getElementById("pos").textContent =
      st.count ? `Slide ${st.index + 1} of ${st.count}` : "No slides";
    document.getElementById("current").textContent = st.slide ? st.slide.text : "";
    document.getElementById("blank").classList.toggle("on", st.blanked);
    const ol = document.getElementById("slides");
    const key = JSON.stringify(slides.map(s => s.label));
    if (key !== listKey) {
      listKey = key;
      ol.replaceChildren(...slides.map((s, i) => {
        const li = document.createElement("li"), b = document.createElement("button");
        b.textContent = `${i + 1}. ${s.label}`;
        b.onclick = () => api(`/slides/${i}/show`, "POST").then(refresh);
        li.append(b);
        return li;
      }));
    }
    [...ol.children].forEach((li, i) => li.classList.toggle("active", i === st.index));
  } catch (e) {
    document.getElementById("current").textContent = e === 401 ? "Unauthorised: add ?token=…" : "Disconnected";
  }
}
const act = (path, body) => () => api(path, "POST", body).then(refresh);
document.getElementById("next").onclick = act("/next");
document.getElementById("prev").onclick = act("/prev");
document.getElementById("blank").onclick = act("/blank", {});
refresh(); setInterval(refresh, 1000);
</script>
</body>
</html>
"""
