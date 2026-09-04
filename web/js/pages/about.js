/* 页面 4：关于软件 */

"use strict";

const PageAbout = {
  enter() {
    const m = App.meta || {};
    document.getElementById("about-author").textContent = m.author || "ALing Studios";
    document.getElementById("about-version").textContent = m.version || "V0.1b";
    document.getElementById("about-builddate").textContent = m.buildDate || "—";
    document.getElementById("about-note").textContent = m.note || "本软件由AI辅助完成";

    const foot = document.getElementById("about-foot");
    foot.innerHTML =
      "界面引擎：Microsoft Edge WebView2<br>" +
      "© " + new Date().getFullYear() + " " + App.esc(m.author || "ALing Studios") + " · What Is My PC";
  },
};
