/* 分享图生成：左侧评级圆超出边界，右上角大字评分，右下角软件名 */

"use strict";

const ShareImage = {
  openModal(grade, total) {
    ShareImage.render(document.getElementById("share-canvas"), grade, total);
    document.getElementById("share-modal").classList.remove("hidden");
  },

  render(canvas, grade, total) {
    const W = 1280, H = 720;
    canvas.width = W;
    canvas.height = H;
    const ctx = canvas.getContext("2d");
    const FONT = '"Segoe UI Variable Display", "Segoe UI", "Microsoft YaHei UI", sans-serif';

    // ── 背景 ──
    const bg = ctx.createLinearGradient(0, 0, W, H);
    bg.addColorStop(0, "#eef6ff");
    bg.addColorStop(1, "#ffffff");
    ctx.fillStyle = bg;
    ctx.fillRect(0, 0, W, H);

    // 右上角装饰圆环
    ctx.strokeStyle = "rgba(47,124,246,.08)";
    ctx.lineWidth = 30;
    ctx.beginPath(); ctx.arc(W - 60, 40, 180, 0, Math.PI * 2); ctx.stroke();
    ctx.lineWidth = 14;
    ctx.beginPath(); ctx.arc(W - 160, 200, 90, 0, Math.PI * 2); ctx.stroke();

    // ── 左侧大圆：超出左边界（含左上、左下角）──
    const cx = 170, cy = H / 2, r = 560;
    ctx.save();
    ctx.shadowColor = "rgba(47,124,246,.25)";
    ctx.shadowBlur = 60;
    const cg = ctx.createLinearGradient(cx - r, cy - r, cx + r, cy + r);
    cg.addColorStop(0, "#cfe7ff");
    cg.addColorStop(0.55, "#a9d4ff");
    cg.addColorStop(1, "#8ec5ff");
    ctx.fillStyle = cg;
    ctx.beginPath(); ctx.arc(cx, cy, r, 0, Math.PI * 2); ctx.fill();
    ctx.restore();

    ctx.strokeStyle = "rgba(255,255,255,.9)";
    ctx.lineWidth = 8;
    ctx.beginPath(); ctx.arc(cx, cy, r - 6, 0, Math.PI * 2); ctx.stroke();

    // ── 圆心评级 ──
    const gradeStr = grade || "D";
    let fontSize = 250;
    if (gradeStr.length === 2) fontSize = 180;
    if (gradeStr.length >= 3) fontSize = 130;
    ctx.fillStyle = "#1d5fc2";
    ctx.font = `800 ${fontSize}px ${FONT}`;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(gradeStr, cx, cy - fontSize * 0.04);

    ctx.font = `600 30px ${FONT}`;
    ctx.fillStyle = "#3c6db3";
    ctx.fillText("评级", cx, cy + fontSize * 0.55);

    // ── 右上：大字评分 ──
    ctx.textAlign = "right";
    ctx.textBaseline = "alphabetic";

    ctx.font = `500 30px ${FONT}`;
    ctx.fillStyle = "#64778c";
    ctx.fillText("我的电脑性能得分", W - 64, 150);

    ctx.font = `800 160px ${FONT}`;
    ctx.fillStyle = "#17273a";
    ctx.fillText(String(total ?? 0), W - 60, 305);

    ctx.font = `700 44px ${FONT}`;
    ctx.fillStyle = "#2f7cf6";
    ctx.fillText("评级 " + gradeStr, W - 64, 375);

    ctx.font = `400 24px ${FONT}`;
    ctx.fillStyle = "#94a5b8";
    const now = new Date();
    const pad = (n) => String(n).padStart(2, "0");
    ctx.fillText(
      `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}`,
      W - 64, 420
    );

    // ── 右下：软件名 ──
    ctx.font = `800 46px ${FONT}`;
    ctx.fillStyle = "#1b2733";
    ctx.fillText("What Is My PC", W - 64, H - 78);

    ctx.font = `400 23px ${FONT}`;
    ctx.fillStyle = "#94a5b8";
    ctx.fillText("ALing Studios · 整机信息与跑分工具", W - 64, H - 40);
  },
};

document.addEventListener("DOMContentLoaded", () => {
  const modal = document.getElementById("share-modal");

  document.getElementById("share-close").addEventListener("click", () =>
    modal.classList.add("hidden")
  );
  modal.addEventListener("click", (e) => {
    if (e.target === modal) modal.classList.add("hidden");
  });

  document.getElementById("share-save").addEventListener("click", async () => {
    const btn = document.getElementById("share-save");
    btn.disabled = true;
    btn.textContent = "保存中…";
    try {
      const canvas = document.getElementById("share-canvas");
      const r = await app_saveShare(canvas.toDataURL("image/png"));
      toast("分享图已保存：" + r.path, "ok", 5000);
      modal.classList.add("hidden");
    } catch (e) {
      toast("保存失败：" + (e && e.message ? e.message : "未知错误"), "err");
    } finally {
      btn.disabled = false;
      btn.textContent = "保存分享图";
    }
  });
});
