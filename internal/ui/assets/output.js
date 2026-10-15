// The output page is a pure function of server state: re-read on every change.
import { api, onState, params } from "/assets/common.js";

const slide = document.getElementById("slide");

function report(name, body) {
  return api("/report/" + name, "POST", body).catch(() => {});
}

async function render() {
  const st = await api("/status");
  slide.textContent = st.slide ? st.slide.text : "";
  document.body.classList.toggle("blank", st.blanked);
  if (params.has("debug")) {
    const r = slide.getBoundingClientRect();
    report("render", {
      text: slide.textContent, rect: [r.x, r.y, r.width, r.height],
      inner: [innerWidth, innerHeight], dpr: devicePixelRatio, visibility: document.visibilityState,
    });
  }
}

if (params.has("debug")) {
  document.body.classList.add("debug");
  const dbg = document.getElementById("dbg");
  dbg.hidden = false;
  const tick = () => { dbg.textContent = `${innerWidth}x${innerHeight} dpr=${devicePixelRatio} ${document.visibilityState} ${new Date().toLocaleTimeString()}`; };
  tick();
  setInterval(tick, 500);
}

onState(render);

// ---- ?probe: what can this webview actually play? (used by `lumen -selftest`) ----
async function playTest(src, alpha) {
  const v = document.createElement("video");
  v.muted = true; v.playsInline = true; v.preload = "auto"; v.src = src;
  const out = { src };
  try {
    await Promise.race([v.play(), new Promise((_, rej) => setTimeout(() => rej(new Error("play() timeout")), 5000))]);
    await new Promise((r) => setTimeout(r, 900));
    out.currentTime = Number(v.currentTime.toFixed(2));
    out.videoWidth = v.videoWidth;
    out.ok = v.currentTime > 0.2 && v.videoWidth > 0;
    if (alpha && out.ok) {
      const c = document.createElement("canvas");
      c.width = v.videoWidth; c.height = v.videoHeight;
      const g = c.getContext("2d");
      g.drawImage(v, 0, 0);
      out.alphaSample = g.getImageData(10, 10, 1, 1).data[3]; // 255 opaque; about 128 means alpha was decoded
    }
  } catch (e) { out.ok = false; out.error = String(e.message || e); }
  if (v.error) out.mediaError = { code: v.error.code, message: v.error.message };
  v.removeAttribute("src"); v.load();
  return out;
}

async function probe() {
  if (document.visibilityState !== "visible") {
    await new Promise((r) => document.addEventListener("visibilitychange", () => document.visibilityState === "visible" && r()));
  }
  const types = {
    "h264+aac (mp4)": 'video/mp4; codecs="avc1.640028,mp4a.40.2"',
    "h264 baseline": 'video/mp4; codecs="avc1.42E01E"',
    "hevc (hvc1)": 'video/mp4; codecs="hvc1.1.6.L93.B0"',
    "av1": 'video/mp4; codecs="av01.0.05M.08"',
    "vp8": 'video/webm; codecs="vp8"',
    "vp9+opus (webm)": 'video/webm; codecs="vp9,opus"',
    "aac audio": 'audio/mp4; codecs="mp4a.40.2"',
    "mp3": "audio/mpeg", "opus (ogg)": 'audio/ogg; codecs="opus"', "flac": "audio/flac",
  };
  const el = document.createElement("video");
  const canPlay = Object.fromEntries(Object.entries(types).map(([k, t]) => [k, el.canPlayType(t) || "no"]));
  let outputs;
  try { outputs = (await navigator.mediaDevices.enumerateDevices()).filter((d) => d.kind === "audiooutput").length; }
  catch (e) { outputs = "error: " + e.message; }
  const playback = {};
  for (const c of ["h264_aac.mp4", "vp9_opus.webm", "hevc.mp4", "av1.mp4"]) playback[c] = await playTest("/media/" + c);
  playback["vp9_alpha.webm"] = await playTest("/media/vp9_alpha.webm", true);
  await report("media", {
    userAgent: navigator.userAgent, canPlayType: canPlay, playback,
    setSinkId: typeof HTMLMediaElement.prototype.setSinkId, audioOutputDevices: outputs,
    page: { dpr: devicePixelRatio, screen: [screen.width, screen.height], inner: [innerWidth, innerHeight], visibility: document.visibilityState },
  });
}

if (params.has("probe")) probe();
