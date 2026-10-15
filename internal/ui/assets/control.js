import { api, onState } from "/assets/common.js";

const $ = (id) => document.getElementById(id);
const showError = (e) => { $("error").textContent = e ? String(e.message || e) : ""; };

async function refresh() {
  try {
    const st = await api("/status");
    $("pos").textContent = st.count ? `Slide ${st.index + 1} of ${st.count}` : "No slides";
    $("cur").textContent = st.slide ? st.slide.text : "";
    $("blank").classList.toggle("on", st.blanked);
    $("slides").replaceChildren(...st.slides.map((s, i) => {
      const li = document.createElement("li");
      const b = document.createElement("button");
      b.type = "button";
      b.textContent = `${i + 1}. ${s.label}`;
      if (i === st.index) b.classList.add("on");
      b.addEventListener("click", () => act(() => api(`/slides/${i}/show`, "POST")));
      li.append(b);
      return li;
    }));
    $("win").textContent = JSON.stringify(await api("/windows"), null, 1);
    showError(null);
  } catch (e) { showError(e); }
}

async function act(fn) {
  try { await fn(); } catch (e) { showError(e); return; }
  refresh();
}

async function initScreens() {
  try {
    for (const s of await api("/screens")) {
      const b = document.createElement("button");
      b.type = "button";
      b.textContent = `${s.id.replace("\\\\.\\", "")} ${s.rect.w}x${s.rect.h} @${Math.round(s.scale * 100)}%${s.primary ? " (primary)" : ""}`;
      b.addEventListener("click", () => act(() => api("/output", "PUT", { screen: s.id })));
      $("screens").append(b);
    }
  } catch (e) { showError(e); }
}

$("prev").addEventListener("click", () => act(() => api("/prev", "POST")));
$("next").addEventListener("click", () => act(() => api("/next", "POST")));
$("blank").addEventListener("click", () => act(() => api("/blank", "POST", {})));
$("float").addEventListener("click", () => act(() => api("/output", "PUT", { screen: null })));
document.addEventListener("keydown", (e) => {
  if (e.key === "ArrowRight" || e.key === "PageDown" || e.key === " ") { e.preventDefault(); $("next").click(); }
  if (e.key === "ArrowLeft" || e.key === "PageUp") { e.preventDefault(); $("prev").click(); }
  if (e.key === "b" || e.key === ".") $("blank").click();
});

initScreens();
onState(refresh);
