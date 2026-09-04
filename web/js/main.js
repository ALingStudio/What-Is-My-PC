/* What Is My PC — 全局框架 */

"use strict";

const App = {
  meta: null,
  pages: {},

  async init() {
    // 首次启动：免责声明与隐私条款
    await App.checkConsent();

    // 路由
    document.querySelectorAll(".nav-item").forEach((el) => {
      el.addEventListener("click", () => App.go(el.dataset.page));
    });

    try {
      App.meta = await app_getMeta();
    } catch (e) {
      App.meta = { name: "What Is My PC", version: "V0.1b", buildDate: "-", author: "ALing Studios" };
    }
    const nv = document.getElementById("nav-version");
    if (nv) nv.textContent = App.meta.version + " · " + App.meta.author;

    // 注册页面
    if (typeof PageConfig !== "undefined") App.pages.config = PageConfig;
    if (typeof PagePerf !== "undefined") App.pages.perf = PagePerf;
    if (typeof PageBench !== "undefined") App.pages.bench = PageBench;
    if (typeof PageAbout !== "undefined") App.pages.about = PageAbout;

    App.go("config");
  },

  /* 首次启动协议：未同意前阻塞全部功能 */
  async checkConsent() {
    let accepted = true;
    try {
      const c = await app_getConsent();
      accepted = !!(c && c.accepted);
    } catch (e) {
      accepted = true; // 读取失败不阻塞用户
    }
    if (accepted) return;

    const mask = document.getElementById("consent-mask");
    mask.classList.remove("hidden");

    mask.querySelectorAll(".consent-link").forEach((a) => {
      a.addEventListener("click", () => app_openURL(a.dataset.url).catch(() => {}));
    });

    await new Promise((resolve) => {
      document.getElementById("consent-accept").addEventListener("click", async () => {
        try { await app_acceptAgreement(); } catch (e) { /* 忽略写入失败 */ }
        mask.classList.add("hidden");
        resolve();
      });
      document.getElementById("consent-decline").addEventListener("click", async () => {
        try { await app_exit(); } catch (e) { /* 忽略 */ }
      });
    });
  },

  current: null,

  go(name) {
    if (!App.pages[name]) return;
    if (App.current === name) return;

    if (App.current && App.pages[App.current] && App.pages[App.current].leave) {
      App.pages[App.current].leave();
    }

    document.querySelectorAll(".nav-item").forEach((el) =>
      el.classList.toggle("active", el.dataset.page === name)
    );
    document.querySelectorAll(".page").forEach((el) =>
      el.classList.toggle("active", el.id === "page-" + name)
    );

    App.current = name;
    if (App.pages[name].enter) App.pages[name].enter();
  },

  /* ── 格式化工具 ── */

  fmtGB(v) {
    if (v == null || isNaN(v)) return "—";
    return (Math.round(v * 10) / 10).toLocaleString() + " GB";
  },

  fmtPct(v) {
    if (v == null || isNaN(v)) return "—";
    return Math.max(0, Math.min(100, Math.round(v)));
  },

  fmtTemp(v) {
    if (v == null || isNaN(v)) return null;
    return Math.round(v) + "°C";
  },

  esc(s) {
    if (s == null) return "";
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  },
};

/* Toast 提示 */
function toast(msg, kind = "", duration = 3200) {
  const wrap = document.getElementById("toast-wrap");
  const el = document.createElement("div");
  el.className = "toast " + kind;
  el.textContent = msg;
  wrap.appendChild(el);
  setTimeout(() => {
    el.style.transition = "opacity .3s";
    el.style.opacity = "0";
    setTimeout(() => el.remove(), 320);
  }, duration);
}

/* 硬件图标库（线性 SVG） */
const HW_ICONS = {
  cpu: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><rect x="6" y="6" width="12" height="12" rx="2"/><rect x="9.5" y="9.5" width="5" height="5" rx="1"/><path d="M9 3v3M15 3v3M9 18v3M15 18v3M3 9h3M3 15h3M18 9h3M18 15h3"/></svg>',
  board: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="8" y="8" width="5" height="5" rx="1"/><path d="M16 8v2M16 12v2M8 16h3M13 16h3"/></svg>',
  ram: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><rect x="3" y="8" width="18" height="8" rx="1.5"/><path d="M7 8v-2M12 8v-2M17 8v-2M6 16v2M10 16v2M14 16v2M18 16v2"/></svg>',
  gpu: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><rect x="3" y="6" width="18" height="11" rx="2"/><circle cx="12" cy="11.5" r="3.2"/><path d="M6 17v2M10 17v2M14 17v2M18 17v2"/></svg>',
  disk: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><rect x="4" y="5" width="16" height="14" rx="2"/><path d="M4 14h16"/><circle cx="8" cy="16.5" r=".6" fill="currentColor"/></svg>',
  net: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><path d="M12 20h.01"/><path d="M8.5 16.4a5 5 0 0 1 7 0"/><path d="M5.6 13.2a9 9 0 0 1 12.8 0"/><path d="M2.8 10a13 13 0 0 1 18.4 0"/></svg>',
  monitor: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><rect x="3" y="4" width="18" height="12" rx="2"/><path d="M9 20h6M12 16v4"/></svg>',
  os: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><rect x="3" y="4" width="18" height="14" rx="2"/><path d="M3 9h18M7 6.5h.01M9.5 6.5h.01"/></svg>',
};
