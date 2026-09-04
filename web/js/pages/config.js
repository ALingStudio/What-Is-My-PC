/* 页面 1：电脑配置 */

"use strict";

const PageConfig = {
  data: null,

  enter() {
    if (PageConfig.data) {
      PageConfig.render(PageConfig.data);
      return;
    }
    PageConfig.load();
  },

  async load() {
    const loading = document.getElementById("config-loading");
    const body = document.getElementById("config-body");
    const errBox = document.getElementById("config-error");

    loading.classList.remove("hidden");
    body.classList.add("hidden");
    errBox.classList.add("hidden");

    try {
      const data = await app_getSystemInfo();
      PageConfig.data = data;
      loading.classList.add("hidden");
      body.classList.remove("hidden");
      PageConfig.render(data);
    } catch (e) {
      loading.classList.add("hidden");
      errBox.classList.remove("hidden");
    }
  },

  /* 清理型号字符串：去掉 (R)/(TM)、主频尾巴等，便于检索 */
  cleanModel(name) {
    if (!name) return "";
    return String(name)
      .replace(/\(R\)|\(TM\)|\(tm\)|\(r\)|®|™/g, "")
      .replace(/CPU\s*@\s*[\d.]+\s*GHz/gi, "")
      .replace(/\s+/g, " ")
      .trim();
  },

  render(d) {
    // ── 顶部居中：电脑名称 ──
    document.getElementById("pc-name").textContent = d.computerName || "未知电脑";
    const osParts = [];
    if (d.os) {
      if (d.os.Caption) osParts.push(d.os.Caption);
      if (d.os.Arch) osParts.push(d.os.Arch);
      if (d.os.Build) osParts.push("Build " + d.os.Build);
    }
    if (d.model) osParts.push(d.model);
    document.getElementById("pc-name-sub").textContent = osParts.join(" · ") || "—";

    // ── 配件卡片（靠右区域）──
    const grid = document.getElementById("config-grid");
    grid.innerHTML = "";

    const mkRow = (key, val, clickQuery) => {
      const isClick = !!clickQuery && !!val;
      return `<div class="hw-row">
        <span class="hw-key">${App.esc(key)}</span>
        <span class="hw-val${isClick ? " clickable" : ""}"${
          isClick ? ` data-query="${App.esc(clickQuery)}" title="点击查看官网信息"` : ""
        }>${App.esc(val == null || val === "" ? "—" : val)}</span>
      </div>`;
    };

    const mkCard = (icon, title, rowsHTML) => {
      const el = document.createElement("div");
      el.className = "card hw-card";
      el.innerHTML = `
        <div class="hw-head">
          <div class="hw-icon">${HW_ICONS[icon] || ""}</div>
          <div class="hw-title">${App.esc(title)}</div>
        </div>
        <div class="hw-rows">${rowsHTML}</div>`;
      grid.appendChild(el);
    };

    // CPU
    if (d.cpu) {
      const c = d.cpu;
      const name = PageConfig.cleanModel(c.Name);
      mkCard("cpu", "处理器 CPU",
        mkRow("名称", name || c.Name, name) +
        mkRow("核心 / 线程", `${c.Cores ?? "—"} 核 / ${c.Threads ?? "—"} 线程`) +
        mkRow("最大主频", c.MaxClockMHz ? (c.MaxClockMHz / 1000).toFixed(1) + " GHz" : null) +
        mkRow("缓存", c.L3KB ? "L3 " + Math.round(c.L3KB / 1024) + " MB" : (c.L2KB ? "L2 " + Math.round(c.L2KB / 1024) + " MB" : null)) +
        mkRow("插槽", c.Socket)
      );
    }

    // 主板
    if (d.board) {
      const q = [d.board.Manufacturer, d.board.Product].filter(Boolean).join(" ");
      mkCard("board", "主板",
        mkRow("制造商", d.board.Manufacturer) +
        mkRow("型号", d.board.Product, q) +
        (d.bios ? mkRow("BIOS", (d.bios.Version || "") + (d.bios.ReleaseDate ? " · " + String(d.bios.ReleaseDate).slice(0, 10) : "")) : "")
      );
    }

    // 内存
    if (d.memory) {
      let rows = mkRow("总容量", App.fmtGB(d.memory.TotalGB));
      (d.memory.Modules || []).forEach((m, i) => {
        const desc = [m.CapacityGB ? m.CapacityGB + "GB" : null, m.Manufacturer, m.PartNumber,
          (m.ConfiguredMHz || m.SpeedMHz) ? (m.ConfiguredMHz || m.SpeedMHz) + "MHz" : null]
          .filter(Boolean).join(" ");
        rows += mkRow("插槽 " + (i + 1) + (m.DeviceLocator ? " · " + m.DeviceLocator : ""), desc || "—");
      });
      mkCard("ram", "内存", rows);
    }

    // 显卡
    if (d.gpus && d.gpus.length) {
      let rows = "";
      d.gpus.forEach((g) => {
        const name = PageConfig.cleanModel(g.Name);
        rows += mkRow("名称", name || g.Name, name);
        rows += mkRow("显存", g.VRAMGB ? App.fmtGB(g.VRAMGB) : null);
        rows += mkRow("驱动版本", g.DriverVersion);
        if (g.Resolution) rows += mkRow("当前分辨率", g.Resolution);
      });
      mkCard("gpu", "显卡 GPU", rows);
    }

    // 磁盘
    if (d.disks && d.disks.length) {
      let rows = "";
      d.disks.forEach((dk) => {
        rows += mkRow("型号", dk.Model, PageConfig.cleanModel(dk.Model));
        rows += mkRow("容量", App.fmtGB(dk.SizeGB));
        rows += mkRow("接口 / 类型", [dk.InterfaceType, dk.MediaType].filter(Boolean).join(" · ") || "—");
      });
      (d.volumes || []).forEach((v) => {
        const used = v.SizeGB && v.FreeGB != null ? Math.round(((v.SizeGB - v.FreeGB) / v.SizeGB) * 100) : null;
        rows += mkRow(`卷 ${v.Drive}${v.Label ? " · " + v.Label : ""}`,
          `${App.fmtGB(v.FreeGB)} 可用 / ${App.fmtGB(v.SizeGB)}${used != null ? "（已用 " + used + "%）" : ""}`);
      });
      mkCard("disk", "磁盘", rows);
    }

    // 网络
    if (d.networkAdapters && d.networkAdapters.length) {
      let rows = "";
      d.networkAdapters.forEach((n) => {
        rows += mkRow("适配器", n.Name, PageConfig.cleanModel(n.Name));
        rows += mkRow("速率", n.SpeedMbps ? n.SpeedMbps + " Mbps" : null);
        rows += mkRow("MAC", n.MAC);
      });
      if (d.ipAddresses && d.ipAddresses.length) {
        rows += mkRow("IPv4", d.ipAddresses.join("，"));
      }
      mkCard("net", "网络", rows);
    }

    // 显示器
    if (d.monitors && d.monitors.length) {
      let rows = "";
      d.monitors.forEach((m) => {
        const q = [m.Manufacturer, m.Name].filter(Boolean).join(" ");
        rows += mkRow("名称", m.Name || m.Manufacturer || "—", q);
        rows += mkRow("制造年份", m.Year ? m.Year + (m.Week ? " 第 " + m.Week + " 周" : "") : null);
      });
      mkCard("monitor", "显示器", rows);
    }

    // 系统
    if (d.os) {
      mkCard("os", "操作系统",
        mkRow("系统", d.os.Caption) +
        mkRow("版本", [d.os.Version, d.os.Build ? "Build " + d.os.Build : null].filter(Boolean).join(" · ")) +
        mkRow("架构", d.os.Arch)
      );
    }

    // ── 官网信息点击 ──
    grid.querySelectorAll(".hw-val.clickable").forEach((el) => {
      el.addEventListener("click", () => PageConfig.openOfficial(el.dataset.query));
    });

    document.getElementById("official-close").onclick = () =>
      document.getElementById("official-panel").classList.add("hidden");
    document.getElementById("config-retry").onclick = () => PageConfig.load();
  },

  async openOfficial(query) {
    const panel = document.getElementById("official-panel");
    const target = document.getElementById("official-target");
    const body = document.getElementById("official-body");

    panel.classList.remove("hidden");
    target.textContent = query;
    body.innerHTML = `<div class="official-loading"><div class="spinner"></div>AI 正在获取部件官方资料…</div>`;
    panel.scrollIntoView({ behavior: "smooth", block: "nearest" });

    try {
      const r = await app_getOfficialInfo(query);
      if (!r || !r.title) throw new Error("empty");
      body.innerHTML = `
        <div class="official-title">${App.esc(r.title)}</div>
        <div class="official-snippet">${App.esc(r.snippet)}</div>
        <div class="official-src">
          <span>来源：${App.esc(r.source || "")}</span>
          ${r.url ? `<a class="official-link" href="javascript:void(0)" data-url="${App.esc(r.url)}">访问页面 →</a>` : ""}
        </div>`;
      const link = body.querySelector(".official-link");
      if (link) {
        link.addEventListener("click", () => app_openURL(link.dataset.url).catch(() => {}));
      }
    } catch (e) {
      body.innerHTML = `<div class="official-fail">资源获取失败</div>`;
    }
  },
};
