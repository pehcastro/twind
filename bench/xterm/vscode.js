const fs = require('fs');
const net = require('net');
const path = require('path');
const zlib = require('zlib');

const CELL = { width: 7, height: 17 };
const VSCODE_BG = [25, 22, 34];
const VSCODE_FG = [204, 204, 204];
const DA1_CONPTY = '\x1b[?61;4c';
const SETTLE_MS = 2500;
const PALETTE_MARK = [255, 0, 255];
const STRAY = /\x1b|\[\?|\[\d+[;A-Za-z]|Gi=|q"1;1|#\d+;2;|�/;

function nodePty() {
  if (process.env.VSCODE_NODE_PTY) return process.env.VSCODE_NODE_PTY;
  const root = path.join(process.env.LOCALAPPDATA, 'Programs', 'Microsoft VS Code');
  const commit = fs.readdirSync(root).find(d => fs.existsSync(path.join(root, d, 'resources')));
  return path.join(root, commit, 'resources', 'app', 'node_modules.asar.unpacked', 'node-pty', 'build', 'Release');
}

function png(width, height, rgb) {
  const chunk = (type, data) => {
    const len = Buffer.alloc(4), crc = Buffer.alloc(4), body = Buffer.concat([Buffer.from(type), data]);
    len.writeUInt32BE(data.length);
    crc.writeUInt32BE(zlib.crc32(body));
    return Buffer.concat([len, body, crc]);
  };
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; header[9] = 2;
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) rgb.copy(raw, y * (width * 3 + 1) + 1, y * width * 3, (y + 1) * width * 3);
  return Buffer.concat([Buffer.from('\x89PNG\r\n\x1a\n', 'latin1'), chunk('IHDR', header), chunk('IDAT', zlib.deflateSync(raw)), chunk('IEND', Buffer.alloc(0))]);
}

function colour(cell, fg) {
  const [isDefault, isRGB, value] = fg ? [cell.isFgDefault(), cell.isFgRGB(), cell.getFgColor()] : [cell.isBgDefault(), cell.isBgRGB(), cell.getBgColor()];
  if (isDefault) return fg ? VSCODE_FG : VSCODE_BG;
  if (isRGB) return [value >> 16 & 255, value >> 8 & 255, value & 255];
  return PALETTE_MARK;
}

const STROKES = { '─': 'lr', '━': 'lr', '│': 'ud', '┃': 'ud', '╭': 'rd', '┌': 'rd', '╮': 'ld', '┐': 'ld', '╰': 'ru', '└': 'ru', '╯': 'lu', '┘': 'lu', '├': 'udr', '┤': 'udl', '┬': 'lrd', '┴': 'lru', '┼': 'lrud' };

function inked(ch, px, py) {
  const code = ch.codePointAt(0), midX = CELL.width >> 1, midY = CELL.height >> 1;
  const strokes = STROKES[ch];
  if (strokes) {
    return px === midX && (strokes.includes('u') && py <= midY || strokes.includes('d') && py >= midY)
      || py === midY && (strokes.includes('l') && px <= midX || strokes.includes('r') && px >= midX);
  }
  if (code === 0x2580) return py < CELL.height / 2;
  if (code === 0x2594) return py < CELL.height / 8;
  if (code >= 0x2581 && code <= 0x2588) return py >= CELL.height * (1 - (code - 0x2580) / 8);
  if (code >= 0x2589 && code <= 0x258f) return px < CELL.width * (0x2590 - code) / 8;
  if (code === 0x2590) return px >= CELL.width / 2;
  if (code === 0x2595) return px >= CELL.width * 7 / 8;
  if (code >= 0x2591 && code <= 0x2593) return (px + py) % (0x2594 - code + 1) === 0;
  return py >= 6 && py < 11 && px >= 1 && px < 6;
}

function frame(term) {
  const buf = term.buffer.active, w = term.cols * CELL.width, h = term.rows * CELL.height, rgb = Buffer.alloc(w * h * 3);
  const lines = [], stray = [];
  let defaults = 0, paletted = 0;
  for (let y = 0; y < term.rows; y++) {
    const line = buf.getLine(buf.viewportY + y);
    lines.push(line.translateToString(false));
    if (STRAY.test(lines[y])) stray.push(y + 1);
    for (let x = 0; x < term.cols; x++) {
      const cell = line.getCell(x);
      if (cell.isBgDefault()) defaults++;
      else if (!cell.isBgRGB()) paletted++;
      const bg = colour(cell, false), fg = colour(cell, true);
      const [back, ink] = cell.isInverse() ? [fg, bg] : [bg, fg];
      const ch = cell.getChars().trim();
      for (let py = 0; py < CELL.height; py++) for (let px = 0; px < CELL.width; px++) {
        const c = ch && inked(ch, px, py) ? ink : back;
        const at = ((y * CELL.height + py) * w + x * CELL.width + px) * 3;
        rgb[at] = c[0]; rgb[at + 1] = c[1]; rgb[at + 2] = c[2];
      }
    }
  }
  return { png: png(w, h, rgb), text: lines.join('\n') + '\n', stray, defaults, paletted };
}

function run(exe, page, cols, rows, images) {
  const { Terminal } = require('@xterm/headless');
  const term = new Terminal({ cols, rows, allowProposedApi: true, windowsPty: { backend: 'conpty' }, windowOptions: { getWinSizePixels: true, getCellSizePixels: true, getWinSizeChars: true } });
  if (images) {
    const warn = console.warn;
    console.warn = (...args) => { if (!String(args[0]).includes('cannot insert output canvas')) warn(...args); };
    class FakeImageData {
      constructor(data, width, height) {
        if (typeof data === 'number') { height = width; width = data; data = new Uint8ClampedArray(width * height * 4); }
        this.data = data; this.width = width; this.height = height;
      }
    }
    const canvas = () => {
      const c = { width: 0, height: 0, classList: { add() {} }, style: {}, remove() {} };
      c.getContext = () => ({ canvas: c, putImageData() {}, createImageData: (w, h) => new FakeImageData(w, h), drawImage() {}, clearRect() {} });
      return c;
    };
    globalThis.ImageData = FakeImageData;
    globalThis.window = globalThis;
    globalThis.document = { createElement: canvas };
    const { ImageAddon } = require('@xterm/addon-image');
    Object.defineProperty(term, 'dimensions', { value: { css: { cell: CELL, canvas: { width: cols * CELL.width, height: rows * CELL.height } } } });
    term.onRender = () => ({ dispose() {} });
    term.loadAddon(new ImageAddon());
  }
  const conpty = require(path.join(nodePty(), 'conpty.node'));
  const env = Object.entries({ ...process.env, TERM_PROGRAM: 'vscode', TERM_PROGRAM_VERSION: '1.140.0', COLORTERM: 'truecolor' })
    .filter(([k]) => !k.startsWith('TWIND_') && k !== 'TERM' && k !== 'WT_SESSION').map(([k, v]) => `${k}=${v}`);
  const pipe = `\\\\.\\pipe\\twind-vscode-${process.pid}`;
  const pty = conpty.startProcess(exe, cols, rows, false, pipe, false, true);
  const conin = new net.Socket({ fd: fs.openSync(pty.conin, 'w'), readable: false, writable: true });
  term.parser.registerCsiHandler({ final: 'c' }, params => {
    if (params.length > 1 || params[0] > 0) return false;
    conin.write(DA1_CONPTY);
    return true;
  });
  const pixels = { 14: `4;${rows * CELL.height};${cols * CELL.width}`, 16: `6;${CELL.height};${CELL.width}` };
  term.parser.registerCsiHandler({ final: 't' }, params => {
    if (params.length !== 1 || !pixels[params[0]]) return false;
    conin.write(`\x1b[${pixels[params[0]]}t`);
    return true;
  });
  term.onData(d => conin.write(d));
  const chunks = [];
  let timer;
  return new Promise(resolve => {
    const done = () => {
      const stream = Buffer.concat(chunks);
      const f = frame(term);
      conpty.kill(pty.pty, true);
      resolve({ ...f, stream });
    };
    const out = net.connect(pty.conout, () => {
      conpty.connect(pty.pty, `"${exe}" docs -page ${page}`, process.cwd(), env, true, () => {});
    });
    out.on('data', d => {
      chunks.push(d);
      term.write(d);
      clearTimeout(timer);
      timer = setTimeout(done, SETTLE_MS);
    });
    out.on('error', () => {});
  });
}

(async () => {
  const [exe, page = 'card', dir = '.', cols = '248', rows = '36', mode = 'off'] = process.argv.slice(2);
  const r = await run(path.resolve(exe), page, +cols, +rows, mode === 'on');
  const name = path.join(dir, `vscode-images-${mode}-${page}`);
  fs.writeFileSync(name + '.png', r.png);
  fs.writeFileSync(name + '.txt', r.text);
  fs.writeFileSync(name + '.bin', r.stream);
  const count = re => (r.stream.toString('latin1').match(re) || []).length;
  console.log(`${page} images ${mode}: ${r.stream.length} B, sixel ${count(/\x1bP[0-9;]*q/g)}, kitty queries ${count(/\x1b_G[^;]*a=q/g)}, kitty images ${count(/\x1b_G(?![^;]*a=q)/g)}, CSI 2J ${count(/\x1b\[2J/g)}, default bg cells ${r.defaults}, palette bg cells ${r.paletted}, rows with stray text ${r.stray.length ? r.stray.join(',') : 'none'}`);
})();
