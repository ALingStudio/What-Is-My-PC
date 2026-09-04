/* 截图画廊：左右滚动 */

"use strict";

document.addEventListener("DOMContentLoaded", () => {
  const track = document.getElementById("carousel-track");
  const prev = document.getElementById("carousel-prev");
  const next = document.getElementById("carousel-next");
  if (!track || !prev || !next) return;

  const step = () => {
    const card = track.querySelector(".screen-card");
    return card ? card.getBoundingClientRect().width + 28 : 600;
  };

  prev.addEventListener("click", () =>
    track.scrollBy({ left: -step(), behavior: "smooth" })
  );
  next.addEventListener("click", () =>
    track.scrollBy({ left: step(), behavior: "smooth" })
  );
});
