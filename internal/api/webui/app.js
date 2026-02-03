function el(id) {
  return document.getElementById(id);
}

function toLines(text) {
  return String(text || "")
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);
}

function pretty(v) {
  try {
    return JSON.stringify(v, null, 2);
  } catch {
    return String(v);
  }
}

function wsURL(path) {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${location.host}${path}`;
}

function buildRunPayload() {
  const platform = el("platform").value;
  const crawlerType = el("crawlerType").value;
  const keywords = el("keywords").value.trim();
  const urls = toLines(el("urls").value);
  const creators = toLines(el("creators").value);

  const payload = {
    platform,
    crawler_type: crawlerType,
  };

  if (crawlerType === "search") {
    payload.keywords = keywords;
  }

  if (crawlerType === "detail") {
    if (platform === "xhs") payload.xhs_specified_note_url_list = urls;
    if (platform === "douyin") payload.dy_specified_note_url_list = urls;
    if (platform === "bilibili") payload.bili_specified_video_url_list = urls;
    if (platform === "weibo") payload.wb_specified_note_url_list = urls;
    if (platform === "tieba") payload.tieba_specified_note_url_list = urls;
    if (platform === "zhihu") payload.zhihu_specified_note_url_list = urls;
    if (platform === "kuaishou") payload.ks_specified_note_url_list = urls;
  }

  if (crawlerType === "creator") {
    if (platform === "xhs") payload.xhs_creator_id_list = creators;
    if (platform === "douyin") payload.dy_creator_id_list = creators;
    if (platform === "weibo") payload.wb_creator_id_list = creators;
    if (platform === "bilibili") payload.bili_creator_id_list = creators;
    if (platform === "tieba") payload.tieba_creator_url_list = creators;
  }

  return payload;
}

async function postJSON(path, body) {
  const res = await fetch(path, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body || {}),
  });
  const data = await res.json().catch(() => ({}));
  return { ok: res.ok, status: res.status, data };
}

async function getJSON(path) {
  const res = await fetch(path, { method: "GET" });
  const data = await res.json().catch(() => ({}));
  return { ok: res.ok, status: res.status, data };
}

let selectedFile = null;

function setFiles(files) {
  const ul = el("files");
  ul.innerHTML = "";
  selectedFile = null;
  el("preview").textContent = "";
  el("btnDownload").setAttribute("href", "#");

  for (const f of files) {
    const li = document.createElement("li");
    li.textContent = `${f.path} (${f.type}, ${f.size} bytes)`;
    li.onclick = () => {
      for (const n of ul.querySelectorAll("li")) n.classList.remove("active");
      li.classList.add("active");
      selectedFile = f;
      el("btnDownload").setAttribute("href", `/data/download/${f.path}`);
    };
    ul.appendChild(li);
  }
}

async function refreshDataFiles() {
  const { ok, data } = await getJSON("/data/files");
  if (!ok) return;
  setFiles(Array.isArray(data.files) ? data.files : []);
}

async function previewSelectedFile() {
  if (!selectedFile) return;
  const { ok, data } = await getJSON(
    `/data/files/${selectedFile.path}?preview=true&limit=50`
  );
  if (!ok) {
    el("preview").textContent = pretty(data);
    return;
  }
  el("preview").textContent = pretty(data);
}

function appendLogLine(line) {
  const logs = el("logs");
  logs.textContent += line;
  if (el("autoScroll").checked) {
    logs.scrollTop = logs.scrollHeight;
  }
}

function connectLogs() {
  const ws = new WebSocket(wsURL("/ws/logs"));
  ws.onmessage = (ev) => appendLogLine(ev.data);
  ws.onerror = () => appendLogLine('{"level":"ERROR","msg":"ws logs error"}\n');
  ws.onclose = () => appendLogLine('{"level":"WARN","msg":"ws logs closed"}\n');
}

function connectStatus() {
  const ws = new WebSocket(wsURL("/ws/status?interval_ms=500"));
  ws.onmessage = (ev) => {
    try {
      const v = JSON.parse(String(ev.data || "").trim());
      el("status").textContent = pretty(v);
    } catch {
      el("status").textContent = String(ev.data || "");
    }
  };
}

async function loadPlatforms() {
  const { ok, data } = await getJSON("/config/platforms");
  const select = el("platform");
  select.innerHTML = "";
  const platforms = ok && Array.isArray(data.platforms) ? data.platforms : [];
  window.__platforms = {};
  for (const p of platforms) {
    const opt = document.createElement("option");
    opt.value = p.key;
    opt.textContent = `${p.key} - ${p.label}`;
    select.appendChild(opt);
    window.__platforms[p.key] = p;
  }
  if (select.options.length > 0) {
    select.value = "xhs";
  }
}

function bindEvents() {
  const updatePayload = () => {
    el("runPayload").textContent = pretty(buildRunPayload());
  };

  const updateModeOptions = () => {
    const platform = el("platform").value;
    const p = (window.__platforms || {})[platform];
    const allowed = p && Array.isArray(p.modes) ? p.modes : ["search", "detail", "creator"];
    const select = el("crawlerType");
    const current = select.value;
    select.innerHTML = "";
    for (const m of allowed) {
      const opt = document.createElement("option");
      opt.value = m;
      opt.textContent = m;
      select.appendChild(opt);
    }
    if (allowed.includes(current)) {
      select.value = current;
    } else if (select.options.length > 0) {
      select.value = select.options[0].value;
    }
  };

  for (const id of ["platform", "crawlerType", "keywords", "urls", "creators"]) {
    el(id).addEventListener("input", updatePayload);
    el(id).addEventListener("change", updatePayload);
  }

  el("platform").addEventListener("change", () => {
    updateModeOptions();
    updatePayload();
  });

  el("btnRun").onclick = async () => {
    const payload = buildRunPayload();
    el("runPayload").textContent = pretty(payload);
    const res = await postJSON("/run", payload);
    appendLogLine(
      `{"level":"INFO","msg":"run","status":${res.status},"ok":${res.ok}}\n`
    );
  };

  el("btnStop").onclick = async () => {
    const res = await postJSON("/stop", {});
    appendLogLine(
      `{"level":"INFO","msg":"stop","status":${res.status},"ok":${res.ok}}\n`
    );
  };

  el("btnLoadData").onclick = refreshDataFiles;
  el("btnPreview").onclick = previewSelectedFile;
  el("btnClearLogs").onclick = () => (el("logs").textContent = "");

  updatePayload();
  updateModeOptions();
}

async function main() {
  await loadPlatforms();
  bindEvents();
  connectLogs();
  connectStatus();
  await refreshDataFiles();
}

main();
