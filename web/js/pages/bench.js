/* 页面 3：跑分测试 */

"use strict";

const BenchState = {
  phase: "idle",            // idle | running | done
  scores: {},               // key -> score（本次）
  runningIndex: -1,
  total: 0,
  grade: "",
  history: null,
};

/* 全局事件入口（Go 侧推送） */
window.__benchEvent = function (ev) {
  if (!ev || !ev.phase) return;

  if (ev.phase === "start") {
    BenchState.phase = "running";
    BenchState.scores = {};
    BenchState.runningIndex = 0;
    BenchState.total = 0;
    BenchState.grade = "";
    if (App.current === "bench") PageBench.renderRunning();
    return;
  }

  if (ev.phase === "item") {
    BenchState.scores[ev.key] = ev.score;
    BenchState.runningIndex = ev.index + 1;
    if (App.current === "bench") PageBench.renderChips();
    return;
  }

  if (ev.phase === "gpu") {
    // 前端执行 WebGL 压力测试后回报
    runGPUTest(4).then((r) => {
      app_gpuResult(r.ok ? Math.round(r.fps) : 0).catch(() => {});
    });
    return;
  }

  if (ev.phase === "done") {
    BenchState.phase = "done";
    BenchState.total = ev.total;
    BenchState.grade = ev.grade;
    if (App.current === "bench") PageBench.renderDone();
    return;
  }
};

const PageBench = {
  items: [],

  async enter() {
    if (!PageBench.items.length && App.meta && App.meta.items) {
      PageBench.items = App.meta.items;
    }
    if (BenchState.phase === "running") {
      PageBench.renderRunning();
    } else if (BenchState.phase === "done") {
      PageBench.renderDone();
    } else {
      PageBench.renderIdle();
    }
  },

  leave() {},

  async loadHistory() {
    try {
      BenchState.history = await app_getHistory();
    } catch (e) {
      BenchState.history = null;
    }
  },

  /* ── 空闲态 ── */

  async renderIdle() {
    await PageBench.loadHistory();
    document.getElementById("bench-frame").classList.remove("done");
    PageBench.setTitle("让我测试你的电脑性能！");
    document.getElementById("bench-total").classList.add("hidden");
    document.getElementById("bench-circle").classList.add("hidden");
    document.getElementById("share-chip").classList.add("hidden");

    const startBtn = document.getElementById("bench-start");
    startBtn.textContent = "开始测试";
    startBtn.disabled = false;
    document.getElementById("bench-hint").textContent =
      "全程约 1~2 分钟，请保持电脑处于空闲状态以获得准确分数";

    const last = BenchState.history && BenchState.history.lastRun;
    PageBench.renderList((key) => {
      const v = last && last.scores ? last.scores[key] : null;
      return v != null ? String(v) : "？";
    }, (key) => (last && last.scores && last.scores[key] != null ? "done" : ""));

    PageBench.renderHistoryStrip();
  },

  setTitle(t) {
    document.getElementById("bench-title").textContent = t;
  },

  renderList(chipText, chipClass) {
    const list = document.getElementById("bench-list");
    list.innerHTML = PageBench.items.map((it, i) => `
      <div class="bench-item" data-key="${App.esc(it.key)}">
        <div class="bench-item-info">
          <div class="bench-item-name">${i + 1}. ${App.esc(it.name)}</div>
          <div class="bench-item-desc">${App.esc(it.desc)}</div>
        </div>
        <div class="score-chip ${chipClass ? chipClass(it.key) : ""}" id="chip-${App.esc(it.key)}">${App.esc(chipText(it.key))}</div>
      </div>`).join("");
  },

  renderHistoryStrip() {
    const el = document.getElementById("bench-history");
    const h = BenchState.history;
    if (!h || (!h.lastRun && !h.bestTotal)) {
      el.textContent = "暂无历史测试记录，跑一次看看吧。";
      return;
    }
    const parts = [];
    if (h.lastRun) parts.push(`上次：${h.lastRun.date} · 总分 ${h.lastRun.total} · 评级 ${h.lastRun.grade}`);
    if (h.bestTotal) parts.push(`最佳：${h.bestDate || ""} · 总分 ${h.bestTotal} · 评级 ${h.bestGrade}`);
    el.textContent = parts.join("　｜　");
  },

  /* ── 测试中 ── */

  renderRunning() {
    document.getElementById("bench-frame").classList.remove("done");
    PageBench.setTitle("我正在仔细地查看并评分");
    document.getElementById("bench-total").classList.add("hidden");
    document.getElementById("bench-circle").classList.add("hidden");
    document.getElementById("share-chip").classList.add("hidden");

    const startBtn = document.getElementById("bench-start");
    startBtn.textContent = "测试中…";
    startBtn.disabled = true;
    document.getElementById("bench-hint").textContent = "正在测试，请勿关闭软件";

    PageBench.renderChips();
  },

  renderChips() {
    PageBench.items.forEach((it, i) => {
      const chip = document.getElementById("chip-" + it.key);
      if (!chip) return;
      const s = BenchState.scores[it.key];
      if (s != null) {
        chip.textContent = s;
        chip.className = "score-chip done";
      } else if (i === BenchState.runningIndex) {
        chip.textContent = "…";
        chip.className = "score-chip running";
      } else {
        chip.textContent = "？";
        chip.className = "score-chip";
      }
    });
  },

  /* ── 完成态 ── */

  renderDone() {
    document.getElementById("bench-frame").classList.add("done");
    PageBench.setTitle("呼哇，完成啦~");

    const totalBox = document.getElementById("bench-total");
    totalBox.classList.remove("hidden");
    document.getElementById("bench-total-num").textContent = BenchState.total;

    // 中央评级圆（穿透大框）
    const circle = document.getElementById("bench-circle");
    circle.classList.remove("hidden");
    document.getElementById("bench-grade").textContent = BenchState.grade;
    document.getElementById("bench-grade-sub").textContent = "评级 · 总分 " + BenchState.total;

    // 分数条全部点亮
    PageBench.items.forEach((it) => {
      const chip = document.getElementById("chip-" + it.key);
      if (!chip) return;
      const s = BenchState.scores[it.key];
      chip.textContent = s != null ? s : "—";
      chip.className = "score-chip done";
    });

    const startBtn = document.getElementById("bench-start");
    startBtn.textContent = "再测一次";
    startBtn.disabled = false;
    document.getElementById("bench-hint").textContent = "";

    document.getElementById("share-chip").classList.remove("hidden");
    PageBench.loadHistory().then(PageBench.renderHistoryStrip);
  },
};

/* 绑定交互（页面静态元素） */
document.addEventListener("DOMContentLoaded", () => {
  document.getElementById("bench-start").addEventListener("click", async () => {
    try {
      const r = await app_startBenchmark();
      if (r && r.ok === false) toast(r.error || "无法开始测试", "err");
    } catch (e) {
      toast("启动测试失败", "err");
    }
  });

  document.getElementById("share-chip").addEventListener("click", () => {
    ShareImage.openModal(BenchState.grade, BenchState.total);
  });
});
