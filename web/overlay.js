// Overlay CBW de um tapete.
//
// Fontes, em ordem de prioridade:
//  1. Mensagem ao vivo do UWW Timing (Mercure http://arena.uww.org/live/{evento}/{tapete}):
//       { state: "starting"|"running", message: { id, idle, weightCategory, round, matchNo,
//         time: "5:55", advantage: "home"|"away"|"",
//         home: { name, country, score, cautions, active }, away: {…} } }
//     Medido em 09/11/2026: publica a cada troca de segundo, não avisa pausa, segundos finais
//     chegam como "0:07" (sem décimos).
//  2. Snapshot REST da luta (scoreboard/fight/open): nomes, países, bandeiras, categoria, fase,
//     durações (roundDuration, roundsNumber, activityTime, injuryTime), tipos de vitória.
//  3. Controles do operador (/control.html): intro, challenge, injury, passivity, victory —
//     estados que o Timing ainda não publicou ao vivo.

import { BOARDS, INTROS, POSITIONS } from '/themes.js';

const params = new URLSearchParams(location.search);
const eventId = params.get('event');
const matName = params.get('mat');
const debug = params.get('debug') === '1';
// still=1: sem transições/animações — mostra direto o estado final (prints, divulgação)
if (params.get('still') === '1') {
  const st = document.createElement('style');
  st.textContent = '*, *::before, *::after { transition: none !important; animation: none !important; }';
  document.head.append(st);
}

const $ = (id) => document.getElementById(id);
const SIDES = ['home', 'away'];
const FINAL_SECONDS = 10;

const state = { snapshot: null, live: null, serverOffset: 0, liveAt: 0, scoreAt: 0,
  settings: { collapseMode: 'auto', afterScoreSeconds: 6, showStartIn: false, showEventInfo: true, background: 'transparent',
    scoreboard: 'classic', position: 'top-start', introPosition: 'auto', introStyle: 'lower', anim: 'auto' } };
const prev = { score: {}, clockText: null, panel: {} };

// ---------- utilitários ----------
const parseTime = (t) => {
  if (t == null || t === '') return null;
  const parts = String(t).trim().split(':').map(Number);
  return parts.some((n) => !Number.isFinite(n)) ? null : parts.reduce((a, n) => a * 60 + n, 0);
};
const fmt = (s) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, '0')}`;
const serverNow = () => Date.now() + state.serverOffset;
const setText = (el, v) => { if (!el) return; const s = v == null ? '' : String(v); if (el.textContent !== s) el.textContent = s; };
const setSrc = (img, code) => {
  const src = code ? `/flags/${String(code).toLowerCase()}.svg` : '';
  if (img.getAttribute('src') !== src) img.setAttribute('src', src);
};
const PHASES = { // nomes curtos do Arena → texto de transmissão
  'Qualif.': 'Qualification', '1/16': '1/16 Final', '1/8': 'Eighth Final', '1/4': 'Quarterfinal',
  '1/2': 'Semifinal', 'Final': 'Final', 'Repech.': 'Repechage', 'Bronze': 'Bronze Medal Match',
};
const phaseName = (r) => PHASES[r] ?? r ?? '';

// Categoria no formato "Seniors | FS | 57".
// Prefere os campos separados do Arena (idade com mais de uma palavra não quebra);
// sem eles, divide o texto curto do Timing/Arena ("Seniors FS 57").
function formatCategory(fight, liveText) {
  const weight = String(fight?.weightCategoryName ?? '').replace(/\s*kg$/i, '').trim();
  const parts = [fight?.audienceName, fight?.sportAlternateName, weight].filter(Boolean);
  if (parts.length === 3) return parts.join(' | ');
  const text = liveText ?? fight?.weightCategoryShortName ?? '';
  return text.trim().split(/\s+/).filter(Boolean).join(' | ');
}

// ---------- modelo único ----------
function model() {
  const snap = state.snapshot;
  const fight = snap?.fight || null;
  const msg = state.live?.message;
  const live = msg && !msg.idle && (!fight || !msg.id || msg.id === fight.id) ? msg : null;
  if (!live && !fight) return null;

  const athlete = (side) => {
    const n = side === 'home' ? 1 : 2;
    const l = live?.[side];
    return {
      name: l?.name ?? fight?.[`fighter${n}DisplayName`] ?? fight?.[`fighter${n}FullName`] ?? '',
      fullName: fight?.[`fighter${n}FullName`] ?? l?.name ?? '',
      country: l?.country ?? fight?.[`team${n}AlternateName`] ?? '',
      score: l?.score ?? 0,
      cautions: l?.cautions ?? 0,
    };
  };

  // Período: o Timing envia o tempo TOTAL restante (começou em 6:00 com 2×3:00).
  const secs = parseTime(live?.time);
  const rd = Number(fight?.roundDuration) || 0;
  const rn = Number(fight?.roundsNumber) || 0;
  let period = null;
  if (secs !== null && rd && rn) period = Math.min(rn, Math.max(1, rn - Math.ceil(secs / rd) + 1));

  const category = formatCategory(fight, live?.weightCategory);

  return {
    home: athlete('home'), away: athlete('away'),
    advantage: live?.advantage || '',
    clockText: live?.time ?? (rd && rn ? fmt(rd * rn) : null),
    secs,
    period,
    category,
    phase: phaseName(live?.round ?? fight?.roundFriendlyName),
    matchNo: live?.matchNo ?? fight?.fightNumber,
    full: rd && rn ? rd * rn : null,
    running: state.live?.state === 'running',
    expectedStart: fight?.expectedStartDate ? Date.parse(fight.expectedStartDate) : null,
    sportName: fight?.sportName, audience: fight?.audienceName,
    weight: fight?.weightCategoryName,
    control: snap?.control || {},
  };
}

// ---------- render ----------
let frame = 0;
const schedule = () => { if (!frame) frame = requestAnimationFrame(render); };

function render() {
  frame = 0;
  const snap = state.snapshot;
  if (snap?.colors) {
    const root = document.documentElement.style;
    if (snap.colors.athlete1) root.setProperty('--c-home', snap.colors.athlete1);
    if (snap.colors.athlete2) root.setProperty('--c-away', snap.colors.athlete2);
  }
  applyBackground();
  applyTheme();
  const m = model();
  const ctl = m?.control || {};
  const intro = ctl.intro;

  // Placar some enquanto a apresentação está no ar (como na transmissão UWW).
  $('bug').classList.toggle('out', !m || !!intro);
  if (m) $('bug').classList.toggle('collapsed', isCollapsed(m, ctl));
  $('intro').classList.toggle('out', !m || !intro);
  if (!m) return;

  setText($('phase'), m.phase);
  setText($('category'), m.category);
  setText($('period'), m.period ? `Period ${m.period}` : '');
  setText($('pn'), m.period || '');
  setText($('bout'), m.matchNo ? `Bout ${m.matchNo}` : '');
  setText($('in-cat'), m.category);
  setText($('in-phase'), m.phase);
  setText($('in-no'), m.matchNo ? `Fight ${m.matchNo}` : '');

  for (const side of SIDES) {
    const a = m[side];
    setSrc($(`f-${side}`), a.country);
    setText($(`n-${side}`), a.name);
    setText($(`c-${side}`), a.country);
    // pontuação com "bump" quando sobe
    const b = $(`s-${side}`).firstElementChild;
    if (b.textContent !== String(a.score)) {
      const up = prev.score[side] != null && a.score > prev.score[side];
      b.textContent = a.score;
      if (up) { b.classList.remove('bump'); void b.offsetWidth; b.classList.add('bump'); state.scoreAt = performance.now(); }
    }
    prev.score[side] = a.score;
    // advertências (cartões)
    const k = $(`k-${side}`);
    if (k.childElementCount !== a.cautions) k.replaceChildren(...Array.from({ length: a.cautions }, () => document.createElement('i')));
    sideEl(side).classList.toggle('adv', m.advantage === side);

    // intro
    setSrc($(`if-${side}`), a.country);
    setText($(`ic-${side}`), a.country);
    setText($(`in-${side}`), a.fullName || a.name);
  }
  const cfg = state.settings;
  setText($('ie-name'), cfg.showEventInfo ? snap?.eventName : '');
  setText($('ie-city'), cfg.showEventInfo ? snap?.eventCity : '');
  setText($('intro-start'), cfg.showStartIn ? startsIn(m.expectedStart) : '');
  setText($('intro-band'), [[m.sportName, m.audience, m.weight].filter(Boolean).join(' - '), m.phase].filter(Boolean).join('   '));

  // relógio
  if (m.clockText != null) {
    const clock = $('clock');
    const changed = clock.textContent !== m.clockText;
    setText(clock, m.clockText);
    const final = m.secs !== null && m.secs > 0 && m.secs <= FINAL_SECONDS;
    clock.classList.toggle('final', final);
    if (final && changed) { clock.classList.remove('tick'); void clock.offsetWidth; clock.classList.add('tick'); }
  }

  renderPanels(m, ctl);
  renderSideBoxes(m, ctl);

  if (debug) {
    $('debug').hidden = false;
    $('debug').textContent = JSON.stringify({ live: state.live, control: ctl, fight: snap?.fight }, null, 2);
  }
}

// Painel lateral: challenge tem prioridade de exibição sobre vitória no mesmo lado.
const CHALLENGE_TEXT = { requested: 'Challenge requested', challenge: 'Challenge', won: 'Challenge won', lost: 'Challenge lost', canceled: 'Challenge canceled' };

function renderPanels(m, ctl) {
  const bug = $('bug');
  const v = ctl.victory;
  bug.classList.toggle('has-winner', !!v);
  for (const side of SIDES) {
    const row = sideEl(side);
    row.classList.toggle('winner', v?.side === side);
    row.classList.toggle('loser', !!v && v.side !== side);

    const panel = $(`p-${side}`);
    let want = null;
    if (ctl.challenge?.side === side) {
      want = { kind: 'challenge', status: ctl.challenge.status || 'challenge', text: CHALLENGE_TEXT[ctl.challenge.status] || 'Challenge' };
    } else if (v?.side === side) {
      want = { kind: 'victory', status: 'victory', text: v.victoryType ? `Winner · ${v.victoryType}` : 'Winner' };
    }
    const key = want ? `${want.kind}:${want.status}:${want.text}` : '';
    if (prev.panel[side] === key) continue;
    prev.panel[side] = key;

    panel.className = 'panel';
    if (!want) continue; // sem .on → desliza de volta
    const a = m[side];
    setText(panel.querySelector('.p-name'), a.name);
    const st = panel.querySelector('.p-status');
    setText(st, want.text);
    st.classList.remove('swap'); void st.offsetWidth; st.classList.add('swap');
    panel.classList.add('on', want.kind === 'victory' ? 'victory' : `st-${want.status}`);
  }
}

// Caixa lateral: atendimento (prioridade) ou passividade, com contagem regressiva.
function renderSideBoxes(m, ctl) {
  const fight = state.snapshot?.fight || {};
  for (const side of SIDES) {
    const box = $(`x-${side}`);
    let cls = '', text = '';
    const inj = ctl.injury?.side === side ? ctl.injury : null;
    const pas = ctl.passivity?.side === side ? ctl.passivity : null;
    // Com o Timing conectado direto, o tempo vem dele (ctl.time) e respeita a pausa;
    // acionado pelo operador, o overlay conta a partir do início.
    if (inj) {
      const total = inj.durationSec || Number(fight.injuryTime) || 240;
      const left = inj.time ? (parseTime(inj.time) ?? 0) : Math.max(0, total - Math.floor((serverNow() - inj.startedAt) / 1000));
      cls = `on injury ${inj.kind === 'blood' ? 'blood' : ''} ${inj.paused ? 'paused' : left <= 30 ? 'urgent' : ''}`;
      text = inj.time || fmt(left);
    } else if (pas) {
      const total = pas.durationSec || Number(fight.activityTime) || 30;
      const left = pas.time ? (parseTime(pas.time) ?? 0) : Math.max(0, total - Math.floor((serverNow() - pas.startedAt) / 1000));
      cls = `on ${pas.paused ? 'paused' : left <= 5 ? 'urgent' : ''}`;
      text = String(Math.floor(left)).padStart(2, '0');
    }
    const want = `side-box ${cls}`.trim();
    if (box.className !== want) box.className = want;
    setText(box, text);
  }
}
// Reavalia periodicamente: cronômetros de atendimento/passividade, pausa (layout automático),
// "Início em". render() só toca no DOM quando algo muda.
setInterval(schedule, 250);

// ---------- encaixe na janela ----------
// O overlay é desenhado em 1920×1080. No OBS/vMix (1920×1080) a escala é 1; numa janela
// menor (navegador do notebook) a tela inteira é reduzida para caber — senão a
// apresentação, que fica embaixo, ficava fora da área visível.
function fit() {
  const k = Math.min(innerWidth / 1920, innerHeight / 1080);
  document.body.style.transform = Math.abs(k - 1) < 0.001 ? '' : `scale(${k})`;
}
addEventListener('resize', fit);
fit();

// ---------- modelos (placar / apresentação / posição / animação) ----------
// ?board=, ?introstyle=, ?pos=, ?anim= na URL têm prioridade sobre as configurações.
const sideEl = (side) => $('bug').querySelector(`[data-side="${side}"]`);
const pick = (param, key, valid, fallback) => { const v = params.get(param) || state.settings[key]; return valid.includes(v) ? v : fallback; };
const mounted = {};
function mount(box, id, models, name) {
  if (mounted[id] === name) return false;
  mounted[id] = name;
  box.innerHTML = models[name].html();
  return true;
}
const LEGACY_POS = { left: 'top-start', center: 'top-center' }; // valores antigos
const POS_IDS = POSITIONS.map((p) => p[0]);
function applyTheme() {
  const board = pick('board', 'scoreboard', Object.keys(BOARDS), 'classic');
  const intro = pick('introstyle', 'introStyle', Object.keys(INTROS), 'lower');
  const norm = (v) => LEGACY_POS[v] || v;
  const b = document.body.dataset;
  const posParam = norm(params.get('pos')), posCfg = norm(state.settings.position);
  b.pos = POS_IDS.includes(posParam) ? posParam : POS_IDS.includes(posCfg) ? posCfg : 'top-start';
  const ipParam = norm(params.get('ipos')), ipCfg = norm(state.settings.introPosition);
  b.ipos = POS_IDS.includes(ipParam) ? ipParam : POS_IDS.includes(ipCfg) ? ipCfg : INTROS[intro].pos; // "auto" = padrão do modelo
  b.anim = pick('anim', 'anim', ['auto', 'fade', 'slide', 'drop', 'zoom', 'wipe'], 'auto');
  b.board = board;
  if (mount($('board'), 'board', BOARDS, board)) for (const k of Object.keys(prev.panel)) delete prev.panel[k];
  mount($('introbox'), 'intro', INTROS, intro);
}

// ---------- fundo (transparente ou chroma key) ----------
// ?bg=green|blue|transparent na URL tem prioridade sobre as configurações.
function applyBackground() {
  const bg = params.get('bg') || state.settings.background || 'transparent';
  const cls = bg === 'green' ? 'chroma-green' : bg === 'blue' ? 'chroma-blue' : '';
  if (document.body.className !== cls) {
    document.body.className = cls;
    // a área fora do quadro (quando a janela é maior/desproporcional) também recebe a cor do chroma
    document.documentElement.style.background = { 'chroma-green': '#00ff00', 'chroma-blue': '#0000ff' }[cls] || 'transparent';
  }
}

// ---------- layout expandido / recolhido ----------
// Automático (padrão): expandido antes da luta, nas paradas, logo após um ponto e com
// challenge/atendimento/vitória no ar; recolhido enquanto o cronômetro corre.
// O Timing não avisa pausa: sem mensagem por mais de PAUSE_MS consideramos parado
// (ele publica a cada segundo enquanto corre; já vimos intervalos de ~2 s em rede ruim).
const PAUSE_MS = 2600;
function isCollapsed(m, ctl) {
  // 1) ajuste momentâneo do operador neste tapete; 2) configuração global
  const forced = ctl.layout?.status
    ?? { never: 'expanded', always: 'collapsed' }[state.settings.collapseMode];
  if (forced === 'expanded') return false;
  if (forced === 'collapsed') return true;
  if (ctl.challenge || ctl.injury || ctl.victory) return false;
  if (!m.running || m.secs === null) return false;
  if (m.full && m.secs >= m.full) return false; // ainda não começou
  const now = performance.now();
  // Timing conectado direto informa a pausa (clockRunning); senão, estimamos pela falta de mensagens.
  const clockRunning = state.live?.clockRunning;
  if (clockRunning === false) return false; // parado
  if (clockRunning === undefined && now - state.liveAt > PAUSE_MS) return false; // parado (estimado)
  if (now - state.scoreAt < state.settings.afterScoreSeconds * 1000) return false; // acabou de pontuar
  return true;
}

// "Início em mm:ss" — só se o horário previsto estiver na próxima hora.
function startsIn(ts) {
  if (!ts) return '';
  const left = Math.floor((ts - serverNow()) / 1000);
  if (left <= 0 || left > 3600) return '';
  return `Início em ${fmt(left)}`;
}

// ---------- dados ----------
function applySnapshot(st) {
  state.snapshot = st;
  if (st.serverTime) state.serverOffset = st.serverTime - Date.now();
  if (st.live) state.live = st.live;
  schedule();
}

async function loadSettings() {
  const r = await fetch('/api/settings');
  if (r.ok) { state.settings = await r.json(); schedule(); }
}

async function loadSnapshot() {
  const r = await fetch(`/api/state/${encodeURIComponent(eventId)}/${encodeURIComponent(matName)}`);
  if (r.ok) applySnapshot(await r.json());
}

function subscribe() {
  const topics = [`cbw/live/${eventId}/${matName}`, 'cbw/settings'].map((t) => `topic=${encodeURIComponent(t)}`).join('&');
  const es = new EventSource(`/hub?${topics}`);
  es.onmessage = (e) => {
    const msg = JSON.parse(e.data);
    if (msg.type === 'settings') { state.settings = msg.settings; schedule(); }
    else if (msg.type === 'snapshot' || msg.type === 'control') applySnapshot(msg.state);
    else if (msg.type === 'live') { state.live = msg.live; state.liveAt = performance.now(); schedule(); }
  };
  es.onopen = () => { loadSettings(); loadSnapshot(); }; // ao (re)conectar, garante o estado completo
}

// ---------- modo demonstração (galeria /preview.html) ----------
// overlay.html?mock=<cenário> usa dados fictícios de /mocks.js, sem Arena nem Timing.
async function startMock(name) {
  const { buildState, SCENARIOS } = await import('/mocks.js');
  const sc = SCENARIOS[name];
  if (!sc) { document.body.textContent = `cenário desconhecido: ${name}`; return; }
  if (params.get('settings') !== 'default') await loadSettings().catch(() => {});
  if (sc.running) setInterval(() => { state.liveAt = performance.now(); }, 400);

  // objeto que os passos (steps) do cenário alteram
  const view = {
    get live() { return state.live; },
    get control() { return state.snapshot.control; },
    set control(v) { state.snapshot = { ...state.snapshot, control: v }; },
  };
  const cycle = () => {
    const st = buildState(name);
    state.snapshot = st.snapshot;
    state.live = st.live;
    state.serverOffset = 0;
    if (sc.running) state.liveAt = performance.now();
    schedule();
    if (!sc.steps) {
      // cenas com cronômetro (passividade/atendimento) recomeçam para não ficarem paradas em 00
      const c = state.snapshot.control || {};
      if (c.passivity || c.injury) setTimeout(cycle, 9000);
      return;
    }
    let last = 0;
    for (const [ms, fn] of sc.steps) {
      last = Math.max(last, ms);
      setTimeout(() => { fn(view); state.snapshot = { ...state.snapshot }; schedule(); }, ms);
    }
    setTimeout(cycle, last + 1800);
  };
  cycle();
}

const mock = params.get('mock');
if (mock) {
  startMock(mock);
} else if (!eventId || !matName) {
  document.body.textContent = 'Informe ?event=<id>&mat=<nome>';
} else {
  Promise.all([loadSettings(), loadSnapshot()]).then(subscribe);
}
