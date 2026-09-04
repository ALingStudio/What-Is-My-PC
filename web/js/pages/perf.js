/* 页面 2：实时占用（参照任务管理器性能页布局） */

"use strict";

const PagePerf = {
  timer: null,
  hist: {},          // 每个资源的历史曲线
  tilesBuilt: false,
  snapCount: 0,
  busy: false,       // 采样进行中标志（防重叠）

  enter() {
    PagePerf.hist = {};
    PagePerf.tilesBuilt = false;
    PagePerf.snapCount = 0;
    document.getElementById("perf-loading").classList.remove("hidden");
    document.getElementById("perf-grid").classList.add("hidden");
    document.getElementById("perf-status").textContent = "正在采样…";
    PagePerf.poll();
    PagePerf.timer = setInterval(PagePerf.poll, 2000);
  },

  leave() {
    if (PagePerf.timer) clearInterval(PagePerf.timer);
    PagePerf.timer = null;
  },

  push(key, v) {
    if (!PagePerf.hist[key]) PagePerf.hist[key] = [];
    const arr = PagePerf.hist[key];
    arr.push(v == null || isNaN(v) ? 0 : v);
    if (arr.length > 60) arr.shift();
  },

  async poll() {
    if (PagePerf.busy) return; // 上次采样未返回则跳过，避免堆叠
    PagePerf.busy = true;
    let snap;
    try {
      snap = await app_getPerf();
    } catch (e) {
      document.getElementById("perf-status").textContent = "采样失败，等待下次重试…";
      PagePerf.busy = false;
      return;
    }
    PagePerf.busy = false;
    PagePerf.snapCount++;

    const memPct = snap.memTotalGB ? (snap.memUsedGB / snap.memTotalGB) * 100 : 0;
    PagePerf.push("cpu", snap.cpuPct);
    PagePerf.push("mem", memPct);
    PagePerf.push("gpu", snap.gpuPct);

    if (!PagePerf.tilesBuilt) {
      PagePerf.buildTiles(snap);
      PagePerf.tilesBuilt = true;
      document.getElementById("perf-loading").classList.add("hidden");
      document.getElementById("perf-grid").classList.remove("hidden");
    }

    PagePerf.updateTiles(snap, memPct);
    document.getElementById("perf-status").textContent =
      "每 1.5 秒刷新 · 第 " + PagePerf.snapCount + " 次采样";
  },

  /* ── 构建卡片骨架 ── */

  buildTiles(snap) {
    const grid = document.getElementById("perf-grid");
    grid.innerHTML = "";

    const mk = (id, wide) => {
      const el = document.createElement("div");
      el.className = "card perf-tile" + (wide ? " wide" : "");
      el.id = id;
      grid.appendChild(el);
      return el;
    };

    // CPU（大卡片）
    mk("tile-cpu", true).innerHTML = `
      <div class="perf-tile-head">
        <div class="perf-tile-name">CPU</div>
        <div class="perf-tile-pct"><span id="cpu-pct">0</span><small> %</small></div>
      </div>
      <div class="spark-wrap"><canvas id="spark-cpu"></canvas></div>
      <div class="perf-tile-foot" id="cpu-foot"></div>`;

    // 内存
    mk("tile-mem", false).innerHTML = `
      <div class="perf-tile-head">
        <div class="perf-tile-name">内存</div>
        <div class="perf-tile-pct"><span id="mem-pct">0</span><small> %</small></div>
      </div>
      <div class="spark-wrap"><canvas id="spark-mem"></canvas></div>
      <div class="perf-tile-foot" id="mem-foot"></div>`;

    // GPU
    mk("tile-gpu", false).innerHTML = `
      <div class="perf-tile-head">
        <div class="perf-tile-name">GPU</div>
        <div class="perf-tile-pct"><span id="gpu-pct">0</span><small> %</small></div>
      </div>
      <div class="spark-wrap"><canvas id="spark-gpu"></canvas></div>
      <div class="perf-tile-foot" id="gpu-foot"></div>`;

    // 每块物理磁盘一个卡片
    (snap.physicalDisks || []).forEach((d) => {
      const id = "tile-disk-" + d.Id;
      mk(id, false).innerHTML = `
        <div class="perf-tile-head">
          <div class="perf-tile-name">磁盘 ${App.esc(d.Id)}<span style="font-weight:400;color:var(--faint);font-size:12px">${App.esc(d.Bus || "")}</span></div>
          <div class="perf-tile-pct"><span id="disk-pct-${d.Id}">0</span><small> %</small></div>
        </div>
        <div class="spark-wrap"><canvas id="spark-disk-${d.Id}"></canvas></div>
        <div class="perf-tile-foot" id="disk-foot-${d.Id}"></div>`;
    });

    // 卷占用卡片
    if ((snap.volumes || []).length) {
      mk("tile-vol", false).innerHTML = `
        <div class="perf-tile-head">
          <div class="perf-tile-name">卷占用</div>
        </div>
        <div class="vol-list" id="vol-list"></div>`;
    }
  },

  /* ── 刷新数据 ── */

  tempBadge(v, label) {
    const t = App.fmtTemp(v);
    if (!t) return `<span class="temp-badge na">${label || "温度"} —</span>`;
    const num = parseFloat(t);
    const cls = num >= 80 ? " hot" : "";
    return `<span class="temp-badge${cls}">${label || "温度"} ${t}</span>`;
  },

  healthBadge(disk, smartPredict) {
    // S.M.A.R.T. 故障预测优先（按磁盘枚举序号匹配，匹配不上则视为全局预警）
    let predictFail = false;
    (smartPredict || []).forEach((s) => {
      if (s.PredictFailure === true) {
        if (s.Index == null || s.Index === disk.Id) predictFail = true;
      }
    });
    if (predictFail) return `<span class="health-badge bad">S.M.A.R.T. 预警</span>`;

    const h = String(disk.Health || "").toLowerCase();
    if (h === "healthy" || h === "ok") return `<span class="health-badge ok">S.M.A.R.T. 正常</span>`;
    if (h === "warning") return `<span class="health-badge warn">S.M.A.R.T. 警告</span>`;
    if (h === "unhealthy" || h === "error") return `<span class="health-badge bad">S.M.A.R.T. 异常</span>`;
    return `<span class="health-badge na">S.M.A.R.T. ${App.esc(disk.Health || "未知")}</span>`;
  },

  updateTiles(snap, memPct) {
    // CPU
    const cpuPct = App.fmtPct(snap.cpuPct);
    document.getElementById("cpu-pct").textContent = cpuPct;
    PagePerf.draw("spark-cpu", PagePerf.hist.cpu, 100);
    let cpuFoot = `<span class="perf-kv">使用率 <b>${cpuPct}%</b></span>`;
    if (snap.zoneTemps && snap.zoneTemps.length) {
      const maxT = Math.max.apply(null, snap.zoneTemps);
      cpuFoot += PagePerf.tempBadge(maxT, "温度");
    } else {
      cpuFoot += `<span class="temp-badge na">温度 —</span>`;
    }
    document.getElementById("cpu-foot").innerHTML = cpuFoot;

    // 内存
    document.getElementById("mem-pct").textContent = App.fmtPct(memPct);
    PagePerf.draw("spark-mem", PagePerf.hist.mem, 100);
    document.getElementById("mem-foot").innerHTML =
      `<span class="perf-kv">已用 <b>${App.fmtGB(snap.memUsedGB)}</b></span>
       <span class="perf-kv">总计 <b>${App.fmtGB(snap.memTotalGB)}</b></span>`;

    // GPU
    document.getElementById("gpu-pct").textContent = App.fmtPct(snap.gpuPct);
    PagePerf.draw("spark-gpu", PagePerf.hist.gpu, 100);
    document.getElementById("gpu-foot").innerHTML = PagePerf.tempBadge(snap.gpuTemp, "GPU 温度");

    // 磁盘
    (snap.physicalDisks || []).forEach((d) => {
      const act = (snap.diskActivity || {})[String(d.Id)] || {};
      const pct = App.fmtPct(act.Pct);
      PagePerf.push("disk-" + d.Id, act.Pct || 0);
      const pctEl = document.getElementById("disk-pct-" + d.Id);
      if (pctEl) pctEl.textContent = pct;
      PagePerf.draw("spark-disk-" + d.Id, PagePerf.hist["disk-" + d.Id], 100);

      const foot = document.getElementById("disk-foot-" + d.Id);
      if (foot) {
        let html = PagePerf.healthBadge(d, snap.smartPredict);
        html += PagePerf.tempBadge(d.TempC);
        html += `<span class="perf-kv">容量 <b>${App.fmtGB(d.SizeGB)}</b></span>`;
        if (act.MBps != null) html += `<span class="perf-kv">吞吐 <b>${act.MBps} MB/s</b></span>`;
        if (d.Wear != null && d.Wear >= 0) html += `<span class="perf-kv">寿命损耗 <b>${d.Wear}%</b></span>`;
        if (d.PowerOnHours) html += `<span class="perf-kv">通电 <b>${Math.round(d.PowerOnHours)} h</b></span>`;
        foot.innerHTML = html;
      }
    });

    // 卷占用
    const volList = document.getElementById("vol-list");
    if (volList && snap.volumes) {
      volList.innerHTML = snap.volumes.map((v) => {
        const usedPct = v.SizeGB ? Math.round(((v.SizeGB - v.FreeGB) / v.SizeGB) * 100) : 0;
        return `<div class="vol-row">
          <span class="vol-name">${App.esc(v.Drive)}</span>
          <div class="vol-bar"><i class="${usedPct > 90 ? "full" : ""}" style="width:${usedPct}%"></i></div>
          <span class="vol-text">${App.fmtGB(v.SizeGB - v.FreeGB)} / ${App.fmtGB(v.SizeGB)}</span>
        </div>`;
      }).join("");
    }
  },

  /* ── 迷你曲线绘制 ── */

  draw(canvasId, data, max) {
    const canvas = document.getElementById(canvasId);
    if (!canvas || !data) return;
    const dpr = window.devicePixelRatio || 1;
    const W = canvas.clientWidth, H = canvas.clientHeight;
    if (W === 0 || H === 0) return;
    if (canvas.width !== W * dpr) canvas.width = W * dpr;
    if (canvas.height !== H * dpr) canvas.height = H * dpr;
    const ctx = canvas.getContext("2d");
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, W, H);

    // 网格
    ctx.strokeStyle = "#eef3f9";
    ctx.lineWidth = 1;
    for (let i = 1; i <= 3; i++) {
      const y = (H / 4) * i;
      ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke();
    }

    const n = data.length;
    if (n < 2) return;
    const step = W / 59; // 固定 60 格，曲线从右侧推进
    const x0 = W - (n - 1) * step;
    const yOf = (v) => H - (Math.max(0, Math.min(max, v)) / max) * (H - 6) - 3;

    ctx.beginPath();
    data.forEach((v, i) => {
      const x = x0 + i * step, y = yOf(v);
      i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
    });
    ctx.strokeStyle = "#2f7cf6";
    ctx.lineWidth = 1.8;
    ctx.lineJoin = "round";
    ctx.stroke();

    // 填充
    ctx.lineTo(x0 + (n - 1) * step, H);
    ctx.lineTo(x0, H);
    ctx.closePath();
    const grad = ctx.createLinearGradient(0, 0, 0, H);
    grad.addColorStop(0, "rgba(47,124,246,.22)");
    grad.addColorStop(1, "rgba(47,124,246,.02)");
    ctx.fillStyle = grad;
    ctx.fill();
  },
};
