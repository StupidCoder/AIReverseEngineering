// Global CRT post-process for the Studio. The per-game viewers render normally; this draws a
// full-screen overlay canvas on top of them and, each frame, captures the active viewer's
// canvas into a texture and runs a physical CRT pipeline over it (NTSC signal -> electron beam
// -> phosphor mask -> glass geometry/glow). The overlay is pointer-events:none, so all the
// viewer's own drag/zoom/rotate interactions pass straight through to it.
//
// The shader is a colleague's "physical model" CRT, lightly adapted: the comparison/raw paths
// are dropped (the filter is simply on or off), and the signal resolution is virtualised (a
// console-like ~240-line signal) so the NTSC/dot-crawl artefacts read correctly over content
// that is actually rendered at display resolution.

const VERT = `
attribute vec2 position;
varying vec2 vUv;
void main() {
  vUv = position * 0.5 + 0.5;
  vUv.y = 1.0 - vUv.y;
  gl_Position = vec4(position, 0.0, 1.0);
}`;

const FRAG = `
#extension GL_OES_standard_derivatives : enable
precision highp float;

uniform sampler2D u_texture;
uniform vec2 u_resolution;
uniform vec2 u_texResolution;
uniform float u_time;

uniform float u_noise;
uniform float u_lumaSmear;
uniform float u_iqBlur;
uniform float u_saturation;
uniform float u_phaseSpeed;

uniform float u_scanlineCount;
uniform float u_verticalScale;
uniform float u_beamFocus;
uniform float u_beamBloom;

uniform float u_maskType;
uniform float u_maskStrength;
uniform float u_tvLines;
uniform float u_curvature;
uniform float u_glow;

varying vec2 vUv;
#define PI 3.14159265359

const mat3 rgb2yiq = mat3(0.299, 0.596, 0.211, 0.587, -0.274, -0.523, 0.114, -0.322, 0.312);
const mat3 yiq2rgb = mat3(1.0, 1.0, 1.0, 0.956, -0.272, -1.106, 0.621, -0.647, 1.703);

vec3 toLinear(vec3 sRGB) { return pow(sRGB, vec3(2.4)); }
vec3 toSRGB(vec3 lin) { return pow(lin, vec3(1.0 / 2.2)); }

vec2 curve(vec2 uv) {
  uv = (uv - 0.5) * 2.0;
  uv *= 1.05;
  uv.x *= 1.0 + pow((abs(uv.y) / 5.0), 2.0) * u_curvature;
  uv.y *= 1.0 + pow((abs(uv.x) / 4.0), 2.0) * u_curvature;
  uv = (uv / 2.0) + 0.5;
  uv = uv * 0.92 + 0.04;
  return uv;
}

float gaussian(float x, float sigma) { return exp(-(x * x) / (2.0 * sigma * sigma)); }

// Domain A: NTSC signal
vec3 sampleNTSC(vec2 uv) {
  vec2 contentUV = (uv - 0.5) * vec2(1.0, 1.0 / u_verticalScale) + 0.5;
  if (contentUV.y < 0.0 || contentUV.y > 1.0) return vec3(0.0);
  vec3 col = texture2D(u_texture, contentUV).rgb;
  if (u_iqBlur <= 0.0 && u_lumaSmear <= 0.0) return col;

  vec3 yiq = rgb2yiq * col;
  float w_total = 1.0;
  vec3 yiq_acc = yiq;
  float pixelX = 1.0 / u_texResolution.x;
  float phase = u_time * 60.0 * u_phaseSpeed;

  for (float i = 1.0; i <= 3.0; i += 1.0) {
    float offset = i * pixelX;
    vec3 left = rgb2yiq * texture2D(u_texture, contentUV - vec2(offset * u_iqBlur, 0.0)).rgb;
    vec3 right = rgb2yiq * texture2D(u_texture, contentUV + vec2(offset * u_iqBlur, 0.0)).rgb;
    float lumaDecay = 2.0 / (u_lumaSmear + 0.001);
    float w_y = exp(-i * i * lumaDecay);
    float w_iq = exp(-i * i * 0.5);
    float crawl = sin(contentUV.x * u_texResolution.x * PI + phase);
    w_iq *= (1.0 + 0.3 * crawl);
    yiq_acc.x += (left.x + right.x) * w_y;
    yiq_acc.yz += (left.yz + right.yz) * w_iq;
    w_total += 2.0 * w_y;
  }

  float finalY = yiq_acc.x / w_total;
  float norm = 1.0 + 2.0 * (exp(-1.0 * 0.5) + exp(-4.0 * 0.5) + exp(-9.0 * 0.5));
  vec3 rgb = yiq2rgb * vec3(finalY, yiq_acc.y / norm, yiq_acc.z / norm);
  vec3 gray = vec3(dot(rgb, vec3(0.299, 0.587, 0.114)));
  rgb = mix(gray, rgb, u_saturation);
  return clamp(rgb, 0.0, 1.0);
}

// Domain B: electron beam
vec3 applyBeam(vec2 uv, vec3 inputLinear) {
  float L = dot(inputLinear, vec3(0.2126, 0.7152, 0.0722));
  float y_dist = fract(uv.y * u_scanlineCount) - 0.5;
  float sigma = u_beamFocus + u_beamBloom * L * 0.15;
  return inputLinear * gaussian(y_dist, sigma);
}

// Domain C: phosphor mask
vec3 calculateMaskSample(vec2 uv, float luma) {
  float x_pixel = uv.x * u_resolution.x;
  float y_pixel = uv.y * u_resolution.y;
  float P = u_resolution.x / u_tvLines * 1.5;

  float stagger = 0.0;
  if (u_maskType > 0.5) {
    float row = floor(y_pixel / P);
    if (mod(row, 2.0) > 0.5) stagger = 0.5;
  }
  float phase = (x_pixel / P) + stagger;
  float delta = fwidth(phase);
  float maskFade = 1.0 - smoothstep(0.5, 1.0, delta);
  if (maskFade < 0.01) return vec3(1.0);

  float r_mask = 0.5 + 0.5 * sin(PI * 2.0 * phase);
  float g_mask = 0.5 + 0.5 * sin(PI * 2.0 * phase + 2.0 * PI / 3.0);
  float b_mask = 0.5 + 0.5 * sin(PI * 2.0 * phase + 4.0 * PI / 3.0);

  float v_mask = 1.0;
  if (u_maskType > 0.5) {
    float y_phase = (y_pixel / P) * PI * 2.0;
    v_mask = 0.5 + 0.5 * sin(y_phase);
    v_mask = mix(1.0, v_mask, 0.5);
  }
  vec3 mask = vec3(r_mask, g_mask, b_mask) * v_mask;
  float washout = 1.0 - luma * 0.6;
  float effectiveStrength = u_maskStrength * maskFade * clamp(washout, 0.0, 1.0);
  return mix(vec3(1.0), mask, effectiveStrength);
}

vec3 applyMask(vec2 uv, vec3 inputLinear) {
  float L = dot(inputLinear, vec3(0.2126, 0.7152, 0.0722));
  vec3 accumMask = vec3(0.0);
  float pixelW = 1.0 / u_resolution.x;
  float pixelH = 1.0 / u_resolution.y;
  for (float x = -0.33; x <= 0.33; x += 0.33)
    for (float y = -0.33; y <= 0.33; y += 0.33)
      accumMask += calculateMaskSample(uv + vec2(x * pixelW, y * pixelH), L);
  return inputLinear * (accumMask / 9.0);
}

void main() {
  vec2 uv = vUv;
  vec2 curvedUV = curve(uv);
  vec3 finalCRT = vec3(0.0);

  if (curvedUV.x >= 0.0 && curvedUV.x <= 1.0 && curvedUV.y >= 0.0 && curvedUV.y <= 1.0) {
    vec3 signalVoltage = sampleNTSC(curvedUV);
    float noise = (fract(sin(dot(curvedUV * u_time, vec2(12.9898, 78.233))) * 43758.5453) - 0.5) * u_noise;
    signalVoltage = max(vec3(0.0), signalVoltage + noise);

    vec3 linearLight = toLinear(signalVoltage);
    linearLight = applyBeam(curvedUV, linearLight);
    linearLight = applyMask(curvedUV, linearLight);

    // glass halation glow
    vec3 glowAcc = vec3(0.0);
    float glowWeight = 0.0;
    float blurStride = 2.5 / u_resolution.x;
    for (float x = -2.0; x <= 2.0; x++) {
      for (float y = -2.0; y <= 2.0; y++) {
        vec2 off = vec2(x, y) * blurStride * 2.0;
        vec2 gUV = (curvedUV + off - 0.5) * vec2(1.0, 1.0 / u_verticalScale) + 0.5;
        vec3 s = vec3(0.0);
        if (gUV.y >= 0.0 && gUV.y <= 1.0) s = toLinear(texture2D(u_texture, gUV).rgb);
        float w = 1.0 / (1.0 + dot(vec2(x, y), vec2(x, y)));
        glowAcc += s * 1.5 * w;
        glowWeight += w;
      }
    }
    vec3 finalLinear = linearLight + (glowAcc / glowWeight) * u_glow;
    float vig = 16.0 * curvedUV.x * curvedUV.y * (1.0 - curvedUV.x) * (1.0 - curvedUV.y);
    finalLinear *= pow(vig, 0.2);
    finalCRT = toSRGB(finalLinear);
  }
  gl_FragColor = vec4(finalCRT, 1.0);
}`;

const DEFAULTS = {
  noise: 0.12, lumaSmear: 0.7, iqBlur: 1.5, saturation: 1.1, phaseSpeed: 0.5,
  beamFocus: 0.55, beamBloom: 2.2, maskType: 1.0, maskStrength: 0.3, tvLines: 400,
  curvature: 0.7, glow: 0.15,
};

export class CRT {
  constructor(stageEl) {
    this.stage = stageEl;
    this.enabled = false;
    this.source = () => []; // set by the host: returns the canvases to capture (front-to-back)
    this.params = { ...DEFAULTS };

    const canvas = document.createElement('canvas');
    canvas.id = 'crt';
    canvas.style.display = 'none';
    stageEl.appendChild(canvas);
    this.canvas = canvas;

    const gl = canvas.getContext('webgl', { alpha: false, antialias: false, depth: false });
    this.gl = gl;
    this.ok = !!gl;
    if (!gl) return;
    gl.getExtension('OES_standard_derivatives');

    const prog = gl.createProgram();
    gl.attachShader(prog, compile(gl, gl.VERTEX_SHADER, VERT));
    gl.attachShader(prog, compile(gl, gl.FRAGMENT_SHADER, FRAG));
    gl.linkProgram(prog);
    gl.useProgram(prog);
    this.prog = prog;

    const buf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]), gl.STATIC_DRAW);
    const loc = gl.getAttribLocation(prog, 'position');
    gl.enableVertexAttribArray(loc);
    gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);

    this.u = {};
    for (const n of ['u_resolution', 'u_texResolution', 'u_time', 'u_noise', 'u_lumaSmear',
      'u_iqBlur', 'u_saturation', 'u_phaseSpeed', 'u_scanlineCount', 'u_verticalScale',
      'u_beamFocus', 'u_beamBloom', 'u_maskType', 'u_maskStrength', 'u_tvLines', 'u_curvature', 'u_glow']) {
      this.u[n] = gl.getUniformLocation(prog, n);
    }

    this.tex = gl.createTexture();
    gl.bindTexture(gl.TEXTURE_2D, this.tex);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);

    this.scratch = document.createElement('canvas');
    this.sctx = this.scratch.getContext('2d');
    this.sctx.imageSmoothingEnabled = false;

    this._raf = null;
    this._loop = this._loop.bind(this);
    window.addEventListener('resize', () => this._resize());
    this._resize();
  }

  set(key, value) { if (key in this.params) this.params[key] = value; }
  reset() { this.params = { ...DEFAULTS }; }

  setEnabled(on) {
    if (!this.ok) return;
    this.enabled = on;
    this.canvas.style.display = on ? 'block' : 'none';
    if (on) { this._resize(); if (!this._raf) this._raf = requestAnimationFrame(this._loop); }
    else if (this._raf) { cancelAnimationFrame(this._raf); this._raf = null; }
  }

  _resize() {
    const ss = Math.min(2, window.devicePixelRatio || 1);
    const w = Math.max(2, Math.round(this.stage.clientWidth * ss));
    const h = Math.max(2, Math.round(this.stage.clientHeight * ss));
    this.canvas.width = w; this.canvas.height = h;
    this.scratch.width = w; this.scratch.height = h;
    this.sctx.imageSmoothingEnabled = false;
    this.gl.viewport(0, 0, w, h);
  }

  _loop(t) {
    this._render(t * 0.001);
    if (this.enabled) this._raf = requestAnimationFrame(this._loop);
  }

  _render(time) {
    const gl = this.gl, w = this.canvas.width, h = this.canvas.height;
    // composite the active viewer's visible canvas(es) into the scratch buffer.
    const sources = this.source();
    if (!sources.length) return;
    this.sctx.clearRect(0, 0, w, h);
    for (const c of sources) {
      if (!c.width || !c.height) continue;
      this.sctx.drawImage(c, 0, 0, w, h);
    }
    gl.bindTexture(gl.TEXTURE_2D, this.tex);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, this.scratch);

    const p = this.params;
    const sigH = 240, sigW = Math.round(sigH * (w / h));
    gl.uniform2f(this.u.u_resolution, w, h);
    gl.uniform2f(this.u.u_texResolution, sigW, sigH);
    gl.uniform1f(this.u.u_time, time);
    gl.uniform1f(this.u.u_noise, p.noise);
    gl.uniform1f(this.u.u_lumaSmear, p.lumaSmear);
    gl.uniform1f(this.u.u_iqBlur, p.iqBlur);
    gl.uniform1f(this.u.u_saturation, p.saturation);
    gl.uniform1f(this.u.u_phaseSpeed, p.phaseSpeed);
    gl.uniform1f(this.u.u_scanlineCount, sigH);
    gl.uniform1f(this.u.u_verticalScale, 1.0);
    gl.uniform1f(this.u.u_beamFocus, p.beamFocus * 0.2);
    gl.uniform1f(this.u.u_beamBloom, p.beamBloom);
    gl.uniform1f(this.u.u_maskType, p.maskType);
    gl.uniform1f(this.u.u_maskStrength, p.maskStrength);
    gl.uniform1f(this.u.u_tvLines, p.tvLines);
    gl.uniform1f(this.u.u_curvature, p.curvature);
    gl.uniform1f(this.u.u_glow, p.glow);
    gl.drawArrays(gl.TRIANGLES, 0, 6);
  }
}

function compile(gl, type, src) {
  const sh = gl.createShader(type);
  gl.shaderSource(sh, src);
  gl.compileShader(sh);
  if (!gl.getShaderParameter(sh, gl.COMPILE_STATUS)) console.error('CRT shader:', gl.getShaderInfoLog(sh));
  return sh;
}
