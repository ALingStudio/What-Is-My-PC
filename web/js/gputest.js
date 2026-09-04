/* GPU 测试：WebGL 重负载片元着色器渲染
 * 用 gl.finish() 强制等待 GPU 完成，统计真实渲染帧率（不受显示器刷新率限制）。
 */

"use strict";

async function runGPUTest(seconds = 4) {
  let canvas = null, gl = null;
  try {
    canvas = document.createElement("canvas");
    canvas.width = 640;
    canvas.height = 360;
    gl = canvas.getContext("webgl", {
      antialias: false,
      depth: false,
      stencil: false,
      alpha: false,
      powerPreference: "high-performance",
    });
    if (!gl) return { ok: false };

    const vsSrc = `
      attribute vec2 aPos;
      varying vec2 vUv;
      void main() {
        vUv = aPos * 0.5 + 0.5;
        gl_Position = vec4(aPos, 0.0, 1.0);
      }`;
    const fsSrc = `
      precision highp float;
      varying vec2 vUv;
      uniform float uT;
      void main() {
        vec3 c = vec3(0.0);
        for (int i = 0; i < 64; i++) {
          float fi = float(i);
          vec2 p = vUv * 3.0 + vec2(sin(uT * 0.7 + fi * 0.31), cos(uT * 0.9 + fi * 0.17));
          c += vec3(sin(p.x * (fi * 0.13 + 1.0)), cos(p.y * (fi * 0.11 + 1.0)), sin((p.x + p.y) * fi * 0.05)) * 0.015;
        }
        gl_FragColor = vec4(0.5 + 0.5 * c, 1.0);
      }`;

    const compile = (type, src) => {
      const s = gl.createShader(type);
      gl.shaderSource(s, src);
      gl.compileShader(s);
      if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) throw new Error("shader");
      return s;
    };
    const prog = gl.createProgram();
    gl.attachShader(prog, compile(gl.VERTEX_SHADER, vsSrc));
    gl.attachShader(prog, compile(gl.FRAGMENT_SHADER, fsSrc));
    gl.linkProgram(prog);
    if (!gl.getProgramParameter(prog, gl.LINK_STATUS)) throw new Error("link");
    gl.useProgram(prog);

    // 全屏两个三角形
    const buf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
    const aPos = gl.getAttribLocation(prog, "aPos");
    gl.enableVertexAttribArray(aPos);
    gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 0, 0);
    const uT = gl.getUniformLocation(prog, "uT");

    // 预热 2 帧
    for (let i = 0; i < 2; i++) {
      gl.uniform1f(uT, i * 0.016);
      gl.drawArrays(gl.TRIANGLES, 0, 3);
      gl.finish();
    }

    const hardCapMs = 10000;
    const deadline = performance.now() + seconds * 1000;
    const start = performance.now();
    let frames = 0;

    while (performance.now() < deadline) {
      const batchEnd = performance.now() + 400;
      while (performance.now() < batchEnd && performance.now() < deadline) {
        gl.uniform1f(uT, frames * 0.016);
        gl.drawArrays(gl.TRIANGLES, 0, 3);
        gl.finish();
        frames++;
      }
      // 让出主线程，避免 UI 完全冻结
      await new Promise((r) => setTimeout(r, 0));
      if (performance.now() - start > hardCapMs) break;
    }

    const elapsed = (performance.now() - start) / 1000;
    const ext = gl.getExtension("WEBGL_lose_context");
    if (ext) ext.loseContext();

    if (frames < 5 || elapsed <= 0) return { ok: false };
    return { ok: true, fps: frames / elapsed };
  } catch (e) {
    return { ok: false };
  } finally {
    try {
      if (gl) {
        const ext = gl.getExtension("WEBGL_lose_context");
        if (ext) ext.loseContext();
      }
    } catch (e) { /* ignore */ }
    canvas = null;
  }
}
