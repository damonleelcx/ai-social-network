// The 3D world behind the landing page: a gate with a voxel heart, two figures walking
// toward it, then a stone terminal among dead trees that the camera pushes into.
//
// Everything renders small and is stretched with nearest-neighbour by CSS. Materials don't
// hold real colours. Green is brightness and a red excess marks "heart" pixels; the final pass
// dithers that onto the theme's ramp, which is why one world can be printed in either theme.

import {
  AdditiveBlending, AmbientLight, BoxGeometry, BufferAttribute, BufferGeometry, CanvasTexture, Color, ColorManagement,
  ConeGeometry, DataTexture, DirectionalLight, Fog, Group, IcosahedronGeometry, InstancedMesh, LinearSRGBColorSpace, Mesh,
  MeshBasicMaterial, MeshLambertMaterial, NearestFilter, Object3D, OrthographicCamera, PerspectiveCamera, PlaneGeometry,
  PointLight, Points, PointsMaterial, RepeatWrapping, Scene, ShaderMaterial, SphereGeometry, Vector3, WebGLRenderTarget,
  WebGLRenderer,
} from './vendor/three.js';

ColorManagement.enabled = false;

// Keep in step with --bg and friends in style.css.
const RAMPS = {
  dark: { ramp: ['#05030c', '#1f1145', '#4c28a8', '#9a78f0', '#f6f1ff'], heartLo: '#a81a3c', heartHi: '#ff4163' },
  light: { ramp: ['#f6f1ff', '#dccfff', '#a98bf5', '#5a2be0', '#1d0f42'], heartLo: '#e8416a', heartHi: '#cf1a46' },
};

const FOV = 38;
const TAN = Math.tan((FOV * Math.PI) / 360);
const CRT_Y = -40;
const SCREEN = { w: 16, h: 10, y: CRT_Y + 1.2, z: 1.32 };

const lum = (v) => new Color(0, v, 0);
const heartColor = (v = 1) => new Color(v, v * 0.5, 0);

function seeded(seed) {
  let s = seed;
  return () => ((s = (s * 16807) % 2147483647) - 1) / 2147483646;
}

function boxes(list, material = new MeshLambertMaterial()) {
  const mesh = new InstancedMesh(new BoxGeometry(1, 1, 1), material, list.length);
  const o = new Object3D();
  list.forEach((b, i) => {
    o.position.set(b.x, b.y, b.z);
    o.scale.set(b.sx, b.sy ?? b.sx, b.sz ?? b.sx);
    if (b.q) o.quaternion.copy(b.q);
    else o.quaternion.identity();
    o.updateMatrix();
    mesh.setMatrixAt(i, o.matrix);
    mesh.setColorAt(i, b.c);
  });
  mesh.frustumCulled = false;
  return mesh;
}

function voxels(rows, palette, size, depth = 1, material) {
  const list = [];
  const w = rows[0].length;
  rows.forEach((row, y) => {
    for (let x = 0; x < w; x++) {
      const c = palette[row[x]];
      if (!c) continue;
      for (let d = 0; d < depth; d++) {
        list.push({ x: (x - w / 2 + 0.5) * size, y: (rows.length - y - 0.5) * size, z: (d - depth / 2 + 0.5) * size, sx: size, c });
      }
    }
  });
  return boxes(list, material);
}

const HEART = ['..XXX...XXX..', '.XXXXX.XXXXX.', 'XXXXXXXXXXXXX', 'XXXXXXXXXXXXX', 'XXXXXXXXXXXXX', '.XXXXXXXXXXX.', '..XXXXXXXXX..', '...XXXXXXX...', '....XXXXX....', '.....XXX.....', '......X......'];

function heart(size) {
  const g = voxels(HEART, { X: heartColor() }, size, 3, new MeshBasicMaterial());
  g.position.y = (-HEART.length * size) / 2;
  const holder = new Group();
  holder.add(g);
  return holder;
}

/* Suki and whoever she's found for you, from behind, holding hands. */
function figures() {
  const suki = [
    '.....HHHH.....', '....HHHHHH....', '...HHHHHHHH...', '...HhHHHCHH...', '...HhHHHHHH...', '...HHHHHhHH...',
    '...HHHHHhHH...', '..JHHHHHHHHJ..', '.JJHHhHHHHHJJ.', '.JJHHhHHHHHJJ.', '.JJJHHHHHHJJJ.', '.JJ.JHHHHJ.JJS',
    '.JJ.JJJJJJ.SSS', '.SS.KKKKKK....', '....KKKKKK....', '...KKKKKKKK...', '...KKKKKKKK...', '....SS..SS....',
    '....SS..SS....', '....SS..SS....', '....BB..BB....', '...BBB..BBB...',
  ];
  const them = [
    '..............', '.....RRRR.....', '....RRRRRR....', '...RRRrRRRR...', '...RRrRRRRR...', '...RRRRRRRR...',
    '....RRRRRR....', '.....SSSS.....', '...TTddddTT...', '.TTTTddddTTTT.', '.TTTTTddTTTTT.', 'STT.TTTTTT.TT.',
    'SSS.TTTTTT.TT.', '....TTTTTT..SS', '....LLLLLL....', '....LLLLLL....', '....LLLLLL....', '....LL..LL....',
    '....LL..LL....', '....LL..LL....', '....BB..BB....', '...BBB..BBB...',
  ];
  const pal = {
    H: lum(0.03), h: lum(0.3), C: lum(1), J: lum(1), K: lum(0.03), S: lum(0.86), B: lum(0.02),
    R: lum(0.03), r: lum(0.32), T: lum(0.7), d: lum(0.42), L: lum(0.03),
  };
  const g = new Group();
  const a = voxels(suki, pal, 0.19, 3, new MeshBasicMaterial());
  const b = voxels(them, pal, 0.19, 3, new MeshBasicMaterial());
  a.position.x = -1.31;
  b.position.x = 1.31;
  g.add(a, b);
  return g;
}

function gate() {
  const r = seeded(11);
  const g = new Group();
  const stones = [];
  const stone = (x, y, z, sx, sy, sz, v) => stones.push({ x, y, z, sx, sy, sz, c: lum(v + (r() - 0.5) * 0.12) });
  for (const side of [-1, 1]) {
    for (let y = 0; y < 7; y++) stone(side * 3.9, y * 0.9 + 0.45, 0, 1.5 - (y % 2) * 0.12, 0.84, 1.5, 0.58);
    stone(side * 3.9, 6.5, 0, 2, 0.5, 2, 0.8);
    stone(side * 3.9, 0.25, 0, 2.2, 0.5, 2.2, 0.72);
    // the outer column, set back
    for (let y = 0; y < 5; y++) stone(side * 6.1, y * 0.9 + 0.45, -1.2, 1.1, 0.84, 1.1, 0.34);
    stone(side * 6.1, 4.8, -1.2, 1.5, 0.4, 1.5, 0.46);
  }
  const R = 3.9;
  for (let i = 0; i <= 18; i++) {
    const a = (i / 18) * Math.PI;
    const big = i % 2 ? 0.92 : 1.12;
    stones.push({
      x: Math.cos(a) * R, y: 6.75 + Math.sin(a) * R, z: 0, sx: big, sy: big, sz: 1.5,
      c: lum(0.56 + (r() - 0.5) * 0.14),
    });
  }
  g.add(boxes(stones));

  // A far castle, only ever a silhouette.
  const towers = [];
  const cones = new Group();
  const coneMat = new MeshLambertMaterial({ color: lum(0.2) });
  for (let i = 0; i < 11; i++) {
    const x = -15 + i * 3 + (r() - 0.5);
    const h = 5 + r() * 8;
    const w = 1.5 + r() * 1.3;
    towers.push({ x, y: h / 2, z: -11 - r() * 5, sx: w, sy: h, sz: w, c: lum(0.14 + r() * 0.08) });
    const cone = new Mesh(new ConeGeometry(w * 0.82, 2.4, 4), coneMat);
    cone.position.set(x, h + 1.2, towers[i].z);
    cone.rotation.y = Math.PI / 4;
    cones.add(cone);
  }
  g.add(boxes(towers), cones);

  // A pool of light behind the figures, so they read as shapes against it.
  const c = document.createElement('canvas');
  c.width = c.height = 64;
  const x = c.getContext('2d');
  const grad = x.createRadialGradient(32, 40, 2, 32, 40, 32);
  grad.addColorStop(0, '#fff');
  grad.addColorStop(0.55, '#666');
  grad.addColorStop(1, '#000');
  x.fillStyle = grad;
  x.fillRect(0, 0, 64, 64);
  const pool = new Mesh(
    new PlaneGeometry(10, 12),
    new MeshBasicMaterial({ map: new CanvasTexture(c), color: lum(0.62), transparent: true, blending: AdditiveBlending, depthWrite: false }),
  );
  pool.position.set(0, 3.8, -0.8);
  g.add(pool);
  return g;
}

/* A bed of flowers: one mound, and a lot of small boxes on it. */
function flowers() {
  const r = seeded(29);
  const g = new Group();
  const mound = new Mesh(new SphereGeometry(1, 16, 8), new MeshLambertMaterial({ color: lum(0.12) }));
  mound.scale.set(11, 2.1, 8);
  mound.position.set(0, -1.75, 3);
  g.add(mound);
  const list = [];
  for (let i = 0; i < 2600; i++) {
    const a = r() * Math.PI * 2;
    const d = Math.sqrt(r());
    const x = Math.cos(a) * d * 11;
    const z = 3 + Math.sin(a) * d * 8;
    const y = -1.75 + 2.1 * Math.sqrt(Math.max(0, 1 - d * d)) + 0.08;
    const s = 0.13 + r() * 0.13;
    const hot = r() < 0.022;
    list.push({ x, y, z, sx: s, sy: s * 0.7, sz: s, c: hot ? heartColor(0.9) : lum(0.3 + r() * 0.5) });
  }
  g.add(boxes(list));
  return g;
}

function tree(list, r, x, y, z, angle, len, thick, depth) {
  const dx = Math.sin(angle) * len;
  const dy = Math.cos(angle) * len;
  const o = new Object3D();
  o.rotation.z = -angle;
  o.updateMatrix();
  list.push({
    x: x + dx / 2, y: y + dy / 2, z, sx: thick, sy: len * 1.04, sz: thick, q: o.quaternion.clone(), c: lum(0.2 + r() * 0.18),
  });
  if (depth === 0) return;
  const n = depth > 3 ? 2 : 2 + (r() > 0.6 ? 1 : 0);
  for (let i = 0; i < n; i++) {
    tree(list, r, x + dx, y + dy, z + (r() - 0.5) * 0.6, angle + (r() - 0.5) * 1.5, len * (0.62 + r() * 0.22), thick * 0.68, depth - 1);
  }
}

/* Cliffs and dead trees: all along the way down, thickest around the terminal. */
function wilds() {
  const r = seeded(7);
  const g = new Group();
  const rockGeo = new IcosahedronGeometry(1, 1);
  const rockMat = new MeshLambertMaterial({ color: lum(0.2), flatShading: true });
  for (let i = 0; i < 46; i++) {
    const side = i % 2 ? 1 : -1;
    const s = 3 + r() * 7;
    const rock = new Mesh(rockGeo, rockMat);
    rock.position.set(side * (24 + r() * 14), 8 - r() * 66, -9 - r() * 16);
    rock.scale.set(s, s * (0.9 + r() * 0.9), s);
    rock.rotation.set(r() * 3, r() * 3, r() * 3);
    g.add(rock);
  }
  const branches = [];
  for (let i = 0; i < 12; i++) {
    const side = i % 2 ? 1 : -1;
    tree(branches, r, side * (12.5 + r() * 10), CRT_Y - 13 + r() * 13, -2 - r() * 7, -side * (0.18 + r() * 0.3), 4.5 + r() * 2.5, 0.42, 5);
  }
  for (let i = 0; i < 6; i++) {
    const side = i % 2 ? 1 : -1;
    tree(branches, r, side * (12 + r() * 6), -8 - r() * 14, -3 - r() * 5, -side * (0.2 + r() * 0.3), 4 + r() * 2, 0.34, 4);
  }
  g.add(boxes(branches));
  return g;
}

function speckleTexture() {
  const r = seeded(3);
  const n = 64;
  const data = new Uint8Array(n * n * 4);
  for (let i = 0; i < n * n; i++) {
    const v = 150 + r() * 105;
    data.set([v, v, v, 255], i * 4);
  }
  const tex = new DataTexture(data, n, n);
  tex.magFilter = tex.minFilter = NearestFilter;
  tex.wrapS = tex.wrapT = RepeatWrapping;
  tex.repeat.set(5, 4);
  tex.needsUpdate = true;
  return tex;
}

function nameplateTexture() {
  const c = document.createElement('canvas');
  c.width = 256;
  c.height = 48;
  const x = c.getContext('2d');
  x.fillStyle = '#000';
  x.fillRect(0, 0, c.width, c.height);
  x.fillStyle = '#fff';
  x.font = '24px "Press Start 2P", monospace';
  x.textBaseline = 'middle';
  x.fillText('SUKI OS', 18, 26);
  const tex = new CanvasTexture(c);
  tex.magFilter = tex.minFilter = NearestFilter;
  tex.generateMipmaps = false;
  return tex;
}

/* The stone terminal. Its screen is a real DOM element; see placeScreen. */
function terminal() {
  const g = new Group();
  g.position.y = CRT_Y;
  const add = (w, h, d, y, z, v, tex) => {
    const m = new Mesh(new BoxGeometry(w, h, d), new MeshLambertMaterial({ color: lum(v), map: tex ?? null }));
    m.position.set(0, y, z);
    g.add(m);
    return m;
  };
  add(21.5, 17, 3, -0.5, -0.6, 0.62, speckleTexture());
  add(19, 12.4, 0.5, 1.2, 0.95, 0.22);
  add(17.8, 11.5, 0.5, 1.2, 1.05, 0.62);
  add(16.7, 10.7, 0.5, 1.2, 1.12, 0.12);
  add(SCREEN.w, SCREEN.h, 0.3, 1.2, SCREEN.z - 0.15, 0.01);

  // the name plate, vents and a little slider
  add(8, 1.7, 0.3, -6.35, 1, 0.22);
  const plate = new Mesh(new PlaneGeometry(7.4, 1.25), new MeshBasicMaterial({ map: nameplateTexture() }));
  plate.position.set(-0, -6.35, 1.17);
  plate.scale.set(1, 1, 1);
  g.add(plate);

  const bits = [];
  for (let i = 0; i < 3; i++) bits.push({ x: 2.6 + 3.2, y: -5.8 - i * 0.45, z: 1, sx: 2.8, sy: 0.18, sz: 0.3, c: lum(0.14) });
  bits.push({ x: 9.1 - 0.4, y: -6.3, z: 1, sx: 2.4, sy: 1, sz: 0.3, c: lum(0.14) });
  bits.push({ x: 9.1 - 0.4, y: -6.1, z: 1.12, sx: 1.8, sy: 0.3, sz: 0.3, c: lum(0.82) });
  // Studs around the rim
  for (let i = 0; i <= 12; i++) {
    const x = -10.2 + i * 1.7;
    bits.push({ x, y: 7.7, z: 1, sx: 0.32, c: lum(1) }, { x, y: -8.7, z: 1, sx: 0.32, c: lum(1) });
  }
  for (let i = 1; i < 9; i++) {
    const y = -8.7 + i * 1.82;
    bits.push({ x: -10.45, y, z: 1, sx: 0.32, c: lum(1) }, { x: 10.45, y, z: 1, sx: 0.32, c: lum(1) });
  }
  g.add(boxes(bits));

  // Heart lanterns either side.
  const lanterns = [];
  for (const side of [-1, 1]) {
    const post = new Mesh(new BoxGeometry(0.5, 6, 0.5), new MeshLambertMaterial({ color: lum(0.5) }));
    post.position.set(side * 12.4, -6, 1.2);
    const cap = new Mesh(new BoxGeometry(1.4, 0.3, 1.4), new MeshLambertMaterial({ color: lum(0.72) }));
    cap.position.set(side * 12.4, -2.95, 1.2);
    const h = heart(0.1);
    h.position.set(side * 12.4, -2.1, 1.2);
    g.add(post, cap, h);
    lanterns.push(h);
  }
  g.userData.lanterns = lanterns;
  return g;
}

function petals() {
  const r = seeded(19);
  const n = 420;
  const pos = new Float32Array(n * 3);
  const speed = new Float32Array(n);
  for (let i = 0; i < n; i++) {
    pos.set([(r() - 0.5) * 64, 22 - r() * 80, -8 + r() * 26], i * 3);
    speed[i] = 0.25 + r() * 0.8;
  }
  const geo = new BufferGeometry();
  geo.setAttribute('position', new BufferAttribute(pos, 3));
  const pts = new Points(geo, new PointsMaterial({ color: lum(0.92), size: 2, sizeAttenuation: false, fog: false }));
  pts.frustumCulled = false;
  pts.userData.speed = speed;
  return pts;
}

const postVert = `varying vec2 vUv; void main(){ vUv = uv; gl_Position = vec4(position.xy, 0.0, 1.0); }`;
const postFrag = `
  uniform sampler2D tScene;
  uniform vec3 uRamp[5];
  uniform vec3 uHeartLo;
  uniform vec3 uHeartHi;
  varying vec2 vUv;
  float bayer(vec2 p) {
    ivec2 i = ivec2(mod(p, 4.0));
    int m[16];
    m[0]=0;m[1]=8;m[2]=2;m[3]=10;m[4]=12;m[5]=4;m[6]=14;m[7]=6;m[8]=3;m[9]=11;m[10]=1;m[11]=9;m[12]=15;m[13]=7;m[14]=13;m[15]=5;
    return (float(m[i.y * 4 + i.x]) + 0.5) / 16.0;
  }
  void main() {
    vec3 c = texture2D(tScene, vUv).rgb;
    float t = bayer(gl_FragCoord.xy);
    vec3 col;
    if (c.r > c.g * 1.25 && c.r > 0.05) {
      col = c.r + (t - 0.5) * 0.35 > 0.6 ? uHeartHi : uHeartLo;
    } else {
      float l = pow(clamp(c.g, 0.0, 1.0), 0.8) * 4.0;
      float idx = min(floor(l + t), 4.0);
      int i = int(idx);
      if (i == 0) col = uRamp[0];
      else if (i == 1) col = uRamp[1];
      else if (i == 2) col = uRamp[2];
      else if (i == 3) col = uRamp[3];
      else col = uRamp[4];
    }
    gl_FragColor = vec4(col, 1.0);
  }`;

const smooth = (a, b, v) => {
  const t = Math.min(1, Math.max(0, (v - a) / (b - a)));
  return t * t * (3 - 2 * t);
};
const mix = (a, b, t) => a + (b - a) * t;

export function createWorld({ canvas, screen, stage, forge, forgeHead, finale, reduced, theme, stepCount, onStep }) {
  const renderer = new WebGLRenderer({ canvas, antialias: false, powerPreference: 'low-power' });
  renderer.outputColorSpace = LinearSRGBColorSpace;
  renderer.setPixelRatio(1);

  const scene = new Scene();
  scene.background = new Color(0, 0, 0);
  scene.fog = new Fog(new Color(0, 0, 0), 22, 80);
  scene.add(new AmbientLight(0xffffff, 1.15));
  const sun = new DirectionalLight(0xffffff, 2.3);
  sun.position.set(-6, 10, 12);
  scene.add(sun);
  const glow = new PointLight(0xffffff, 40, 26);
  glow.position.set(0, 7, 3);
  scene.add(glow);

  const heartMesh = heart(0.24);
  heartMesh.position.set(0, 7.1, 0.2);
  const walkers = figures();
  walkers.position.set(0, 0.32, 4.2);
  const crt = terminal();
  const motes = petals();
  scene.add(gate(), flowers(), wilds(), heartMesh, walkers, crt, motes);

  const camera = new PerspectiveCamera(FOV, 1, 0.5, 140);
  const target = new WebGLRenderTarget(4, 4, { minFilter: NearestFilter, magFilter: NearestFilter });

  const uniforms = {
    tScene: { value: target.texture },
    uRamp: { value: RAMPS[theme].ramp.map((c) => new Color(c)) },
    uHeartLo: { value: new Color(RAMPS[theme].heartLo) },
    uHeartHi: { value: new Color(RAMPS[theme].heartHi) },
  };
  const postScene = new Scene();
  const postCam = new OrthographicCamera(-1, 1, 1, -1, 0, 1);
  postScene.add(new Mesh(new PlaneGeometry(2, 2), new ShaderMaterial({ vertexShader: postVert, fragmentShader: postFrag, uniforms, depthTest: false })));

  // Layout, measured whenever the page changes shape.
  let W = 1;
  let H = 1;
  let stageX = 0;
  let stageY = 0;
  let heroD = 24;
  let forgeTop = 0;
  let forgeRange = 1;
  let flat = false;
  let dirty = true;

  function measure() {
    W = innerWidth;
    H = innerHeight;
    const px = W < 800 ? 2 : 3;
    renderer.setSize(Math.ceil(W / px), Math.ceil(H / px), false);
    target.setSize(Math.ceil(W / px), Math.ceil(H / px));
    camera.aspect = W / H;
    flat = document.documentElement.classList.contains('flat-crt');
    crt.visible = !flat;

    const s = stage.getBoundingClientRect();
    stageX = s.left + s.width / 2;
    stageY = s.top + scrollY + s.height / 2;
    // Fit the gate (about 13 tall, 15 wide) to whatever room the layout gave it.
    const byH = (13.5 * (H / Math.max(s.height, 200))) / (2 * TAN);
    const byW = (16 * (W / Math.max(s.width, 200))) / (2 * TAN * camera.aspect);
    heroD = Math.min(Math.max(byH, Math.min(byW, byH * 1.25)), 60);

    const f = forge.getBoundingClientRect();
    forgeTop = f.top + scrollY;
    forgeRange = Math.max(1, f.height - H);
    dirty = true;
  }

  const eye = new Vector3();
  const look = new Vector3();
  let mx = 0;
  let my = 0;
  let tx = 0;
  let ty = 0;

  function pose() {
    const y = scrollY;
    const toCrt = smooth(H * 0.25, Math.max(H * 0.5, forgeTop), y);
    const heroShift = 1 - smooth(0, H, y);

    // Hero: the gate, centred on the stage.
    const hx = 0;
    const hy = 4.7;
    // Far from the terminal, with room above it for the heading.
    const farD = Math.max(45, 26 / camera.aspect);
    const cy = SCREEN.y + 6.5;
    let ex = mix(hx, 0, toCrt);
    let ey = mix(hy, cy, toCrt);
    let ez = mix(heroD, farD, toCrt);
    let ly = mix(hy, cy, toCrt);

    // Forge: push in until the screen fills the view.
    const p = Math.min(1, Math.max(0, (y - forgeTop) / forgeRange));
    const push = smooth(0, 0.28, p);
    if (!flat) {
      const fit = Math.max((SCREEN.h / 0.8) / (2 * TAN), (SCREEN.w / 0.95) / (2 * TAN * camera.aspect)) + SCREEN.z;
      // Sit a little low, clear of the sticky nav.
      const clear = SCREEN.y + (32 / H) * 2 * (fit - SCREEN.z) * TAN;
      ey = mix(ey, clear, push);
      ly = mix(ly, clear, push);
      ez = mix(ez, fit, push);
      if (p > 0) {
        const chapter = Math.min(stepCount - 1, Math.floor((Math.max(0, p - 0.3) / 0.7) * stepCount));
        onStep(chapter);
      }
      forgeHead.style.opacity = String(1 - smooth(0.02, 0.16, p));
      forgeHead.style.visibility = p > 0.17 ? 'hidden' : 'visible';
    }

    // The way out: back to the heart, seen from low down in the flowers.
    const fr = finale.getBoundingClientRect();
    const end = smooth(H * 0.95, H * 0.1, fr.top);
    const atEnd = fr.top < H;
    if (atEnd) {
      ex = mix(9, 6.5, end);
      ey = 2;
      ez = mix(22, 19, end);
      ly = 5.2;
    }

    const wobble = reduced ? 0 : 1;
    eye.set(ex + mx * (1.2 - push * 0.8) * wobble, ey - my * (0.7 - push * 0.45) * wobble, ez);
    look.set(atEnd ? -3 : 0, ly, 0);
    camera.position.copy(eye);
    camera.lookAt(look);
    // Fog starts just past whatever we're looking at, however far back the layout put us.
    scene.fog.near = Math.max(22, ez + 2);
    scene.fog.far = scene.fog.near + 58;

    // The hero's vanishing point sits over the stage, not the viewport's centre.
    const sx = (stageX - W / 2) * heroShift * (atEnd ? 0 : 1);
    const sy = (stageY - H / 2) * heroShift * (atEnd ? 0 : 1);
    if (sx || sy) camera.setViewOffset(W, H, -sx, -sy, W, H);
    else camera.clearViewOffset();
    camera.updateProjectionMatrix();
    camera.updateMatrixWorld();

    return { p, atEnd, inForge: y + H > forgeTop - H * 0.7 && y < forgeTop + forgeRange + H * 0.9 };
  }

  // Warp the DOM screen onto the 3D one: a homography from its rectangle to the
  // four projected corners, so its text stays sharp, selectable and clickable.
  const corner = new Vector3();
  function project(x, y) {
    corner.set(x, y, SCREEN.z).project(camera);
    return [((corner.x + 1) / 2) * W, ((1 - corner.y) / 2) * H];
  }
  function placeScreen(show) {
    if (flat) return;
    screen.style.visibility = show ? 'visible' : 'hidden';
    if (!show) return;
    const hw = SCREEN.w / 2;
    const hh = SCREEN.h / 2;
    const [x0, y0] = project(-hw, SCREEN.y + hh);
    const [x1, y1] = project(hw, SCREEN.y + hh);
    const [x2, y2] = project(hw, SCREEN.y - hh);
    const [x3, y3] = project(-hw, SCREEN.y - hh);
    const w = screen.offsetWidth;
    const h = screen.offsetHeight;
    const dx1 = x1 - x2;
    const dx2 = x3 - x2;
    const sx = x0 - x1 + x2 - x3;
    const dy1 = y1 - y2;
    const dy2 = y3 - y2;
    const sy = y0 - y1 + y2 - y3;
    const den = dx1 * dy2 - dx2 * dy1;
    const g = (sx * dy2 - dx2 * sy) / den;
    const k = (dx1 * sy - sx * dy1) / den;
    const a = x1 - x0 + g * x1;
    const b = x3 - x0 + k * x3;
    const d = y1 - y0 + g * y1;
    const e = y3 - y0 + k * y3;
    screen.style.transform = `matrix3d(${[a / w, d / w, 0, g / w, b / h, e / h, 0, k / h, 0, 0, 1, 0, x0, y0, 0, 1].join(',')})`;
  }

  // Theme: slide the ramp rather than cut, so the toggle feels like the light changing.
  let themeFrom = null;
  let themeTo = RAMPS[theme];
  let themeStart = 0;
  function setTheme(next) {
    themeFrom = {
      ramp: uniforms.uRamp.value.map((c) => c.clone()),
      lo: uniforms.uHeartLo.value.clone(),
      hi: uniforms.uHeartHi.value.clone(),
    };
    themeTo = RAMPS[next];
    themeStart = performance.now();
    dirty = true;
  }
  function stepTheme(now) {
    if (!themeFrom) return;
    const t = reduced ? 1 : Math.min(1, (now - themeStart) / 420);
    const q = Math.round(t * 7) / 7;
    uniforms.uRamp.value.forEach((c, i) => c.copy(themeFrom.ramp[i]).lerp(new Color(themeTo.ramp[i]), q));
    uniforms.uHeartLo.value.copy(themeFrom.lo).lerp(new Color(themeTo.heartLo), q);
    uniforms.uHeartHi.value.copy(themeFrom.hi).lerp(new Color(themeTo.heartHi), q);
    dirty = true;
    if (t === 1) themeFrom = null;
  }

  function animate(time) {
    const s = time / 1000;
    heartMesh.rotation.y = Math.sin(s * 0.7) * 0.55;
    const beat = 1 + Math.max(0, Math.sin(s * 2.6)) * 0.07;
    heartMesh.scale.setScalar(beat);
    heartMesh.position.y = 7.1 + Math.sin(s * 1.2) * 0.14;
    glow.intensity = 34 + Math.sin(s * 2.6) * 10;
    walkers.position.y = 0.32 + (Math.round(Math.abs(Math.sin(s * 1.4)) * 2) / 2) * 0.04;
    for (const [i, l] of crt.userData.lanterns.entries()) {
      l.scale.setScalar(1 + Math.round(Math.sin(s * 7 + i * 2) * 2) * 0.05);
    }
    const pos = motes.geometry.attributes.position;
    const speed = motes.userData.speed;
    for (let i = 0; i < pos.count; i++) {
      let y = pos.getY(i) - speed[i] * 0.022;
      if (y < -58) y = 22;
      pos.setY(i, y);
      pos.setX(i, pos.getX(i) + Math.sin(s + i) * 0.006);
    }
    pos.needsUpdate = true;
  }

  function frame(now) {
    requestAnimationFrame(frame);
    if (document.hidden) return;
    stepTheme(now);
    if (reduced && !dirty) return;
    mx += (tx - mx) * 0.06;
    my += (ty - my) * 0.06;
    const { atEnd, inForge } = pose();
    placeScreen(inForge && !atEnd);
    // Solid sections cover the canvas between the terminal and the finale.
    const covered = !atEnd && scrollY > forgeTop + forgeRange + H * 1.05;
    if (!covered) {
      if (!reduced) animate(now);
      renderer.setRenderTarget(target);
      renderer.render(scene, camera);
      renderer.setRenderTarget(null);
      renderer.render(postScene, postCam);
    }
    dirty = false;
  }

  addEventListener('resize', measure);
  addEventListener('scroll', () => (dirty = true), { passive: true });
  addEventListener('pointermove', (e) => {
    if (e.pointerType === 'touch') return;
    tx = (e.clientX / innerWidth - 0.5) * 2;
    ty = (e.clientY / innerHeight - 0.5) * 2;
    dirty = true;
  }, { passive: true });
  new ResizeObserver(measure).observe(document.body);
  document.fonts?.ready.then(measure);

  measure();
  requestAnimationFrame(frame);

  return {
    setTheme,
    scrollToStep(i) {
      const p = 0.3 + (0.7 * (i + 0.5)) / stepCount;
      scrollTo({ top: forgeTop + forgeRange * p, behavior: reduced ? 'auto' : 'smooth' });
    },
  };
}
