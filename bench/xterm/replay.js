const fs = require('fs');
const path = require('path');
const { performance } = require('perf_hooks');

class FakeImageData {
  constructor(data, width, height) {
    if (typeof data === 'number') { height = width; width = data; data = new Uint8ClampedArray(width * height * 4); }
    this.data = data; this.width = width; this.height = height;
  }
}
function fakeCanvas() {
  const c = { width: 0, height: 0, classList: { add() {} }, style: {}, remove() {} };
  c.getContext = () => ({ canvas: c, putImageData() {}, createImageData: (w, h) => new FakeImageData(w, h), drawImage() {}, clearRect() {} });
  return c;
}
const warn = console.warn;
const harness = ['cannot insert output canvas', 'writeSync is unreliable'];
console.warn = (...args) => { if (!harness.some(h => String(args[0]).includes(h))) warn(...args); };
globalThis.ImageData = FakeImageData;
globalThis.window = globalThis;
globalThis.document = { createElement: () => fakeCanvas() };

const { Terminal } = require('@xterm/headless');
const { ImageAddon } = require('@xterm/addon-image');

const ALT_SCREEN = '\x1b[?1049h';
const CELL = { width: 7, height: 17 };

function records(file) {
  const all = fs.readFileSync(file), out = [];
  for (let at = 0; at < all.length;) {
    const n = all.readUInt32LE(at);
    out.push(all.subarray(at + 4, at + 4 + n));
    at += 4 + n;
  }
  return out;
}

function write(term, data) {
  const start = performance.now();
  term._core.writeSync(data);
  return performance.now() - start;
}

function drawCalls(term, rows) {
  const buf = term._core.buffer;
  let calls = 0;
  for (let y = rows[0]; y <= rows[1]; y++) {
    const line = buf.lines.get(y + buf.ydisp);
    let last = null;
    for (let x = 0; x < term.cols; x++) {
      const e = line.getExtended(x)?.payload;
      const id = e && e.imageId !== undefined && e.imageId !== -1 && e.tileId !== -1 ? e : null;
      if (id && !(last && last.imageId === id.imageId && last.tileId + 1 === id.tileId)) calls++;
      last = id;
    }
  }
  return calls;
}

function quantile(v, q) {
  const s = [...v].sort((a, b) => a - b);
  return s[Math.min(s.length - 1, Math.floor(s.length * q))];
}

async function replay(file, cols, rows) {
  const term = new Terminal({ cols, rows, allowProposedApi: true });
  Object.defineProperty(term, 'dimensions', { value: { css: { cell: CELL, canvas: { width: cols * CELL.width, height: rows * CELL.height } } } });
  term.onRender = () => ({ dispose() {} });
  const addon = new ImageAddon();
  term.loadAddon(addon);
  await new Promise(r => setTimeout(r, 200));
  const tracker = term._core._inputHandler._dirtyRowTracker;
  let dirty = null;
  const touch = (a, b) => { dirty = dirty ? [Math.min(dirty[0], a), Math.max(dirty[1], b)] : [a, b]; };
  for (const name of ['markDirty', 'markRangeDirty', 'markAllDirty']) {
    const original = tracker[name].bind(tracker);
    tracker[name] = (a, b) => {
      if (name === 'markAllDirty') touch(0, rows - 1);
      else touch(Math.max(0, Math.min(a, rows - 1)), Math.max(0, Math.min(b ?? a, rows - 1)));
      return original(a, b);
    };
  }
  let added = 0;
  const storage = addon._storage, add = storage.addImage.bind(storage);
  storage.addImage = (...args) => { added++; return add(...args); };
  const [startup, ...steps] = records(file);
  write(term, Buffer.concat([Buffer.from(ALT_SCREEN), startup]));
  const ms = [], images = [], rowsDirty = [], calls = [], bytes = [];
  for (const r of steps) {
    added = 0; dirty = null;
    ms.push(write(term, r));
    images.push(added);
    bytes.push(r.length);
    rowsDirty.push(dirty ? dirty[1] - dirty[0] + 1 : 0);
    calls.push(dirty ? drawCalls(term, dirty) : 0);
  }
  const mean = v => v.reduce((a, b) => a + b, 0) / v.length;
  return { name: path.basename(file, '.bin'), p50: quantile(ms, 0.5), p95: quantile(ms, 0.95), bytes: mean(bytes), images: mean(images), rows: mean(rowsDirty), draws: mean(calls), live: storage._images.size };
}

(async () => {
  const [dir, cols = '248', rows = '36'] = process.argv.slice(2);
  for (const f of fs.readdirSync(dir).filter(f => f.endsWith('.bin')).sort()) {
    if (f.startsWith('zed')) continue;
    const r = await replay(path.join(dir, f), +cols, +rows);
    console.log(`${r.name.padEnd(22)} parse p50 ${r.p50.toFixed(2)} ms p95 ${r.p95.toFixed(2)} ms, ${r.bytes.toFixed(0)} B, ${r.images.toFixed(1)} images, ${r.rows.toFixed(1)} rows refreshed, ${r.draws.toFixed(1)} image draws, ${r.live} images live`);
  }
})();
