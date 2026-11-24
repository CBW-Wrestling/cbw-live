// Componentes comuns das telas de operação: cabeçalho com navegação,
// ícones, toasts e utilitários. Sem dependências externas.

export const ICONS = {
  grid: '<path d="M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z"/>',
  gallery: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m3 16 5-5 4 4 3-3 6 6"/><circle cx="15.5" cy="8.5" r="1.5"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>',
  tv: '<rect x="2" y="5" width="20" height="13" rx="2"/><path d="M8 21h8M12 18v3"/>',
  sliders: '<path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6"/>',
  copy: '<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1"/>',
  external: '<path d="M14 3h7v7M10 14 21 3M21 14v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h6"/>',
  radio: '<circle cx="12" cy="12" r="2"/><path d="M16.2 7.8a6 6 0 0 1 0 8.4M7.8 16.2a6 6 0 0 1 0-8.4M19.1 4.9a10 10 0 0 1 0 14.2M4.9 19.1a10 10 0 0 1 0-14.2"/>',
  flag: '<path d="M4 22V4M4 4h13l-2 4 2 4H4"/>',
  shield: '<path d="M12 2 4 5v6c0 5 3.4 9.4 8 11 4.6-1.6 8-6 8-11V5z"/>',
  drop: '<path d="M12 2.7S5 10 5 14.5a7 7 0 0 0 14 0C19 10 12 2.7 12 2.7z"/>',
  timer: '<circle cx="12" cy="13" r="8"/><path d="M12 9v4l2 2M9 2h6"/>',
  trophy: '<path d="M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0zM17 5h3a3 3 0 0 1-3 4M7 5H4a3 3 0 0 0 3 4"/>',
  play: '<path d="M7 4v16l13-8z"/>',
  layout: '<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M9 21V9"/>',
  eraser: '<path d="m7 21-4-4 11-11 7 7-7 7zM10 11l7 7M7 21h14"/>',
  refresh: '<path d="M21 12a9 9 0 1 1-2.6-6.4M21 3v6h-6"/>',
  check: '<path d="m5 12 5 5 9-10"/>',
  mat: '<rect x="3" y="3" width="18" height="18" rx="9"/><circle cx="12" cy="12" r="4"/>',
};

export function icon(name, cls = '') {
  return `<svg class="${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICONS[name] || ''}</svg>`;
}

const NAV = [
  { href: '/', label: 'Tapetes', icon: 'grid', key: 'mats' },
  { href: '/preview.html', label: 'Galeria', icon: 'gallery', key: 'gallery' },
  { href: '/settings.html', label: 'Configurações', icon: 'settings', key: 'settings' },
];

/** Monta o cabeçalho padrão em <header id="app-header">. */
export function mountHeader(active, extraRight = '') {
  const el = document.getElementById('app-header');
  if (!el) return;
  el.className = 'app-header';
  el.innerHTML = `
    <a class="brand" href="/" aria-label="CBW Live — início">
      <img src="/assets/brand/cbw-live-dark.svg" alt="">
      <span><b>CBW Live</b><small>Gráficos de transmissão</small></span>
    </a>
    <nav class="nav" aria-label="Principal">
      ${NAV.map((n) => `<a href="${n.href}" ${n.key === active ? 'aria-current="page"' : ''}>${icon(n.icon)}${n.label}</a>`).join('')}
    </nav>
    <div class="header-right">
      ${extraRight}
      <span class="badge ok live" id="conn-badge"><span class="dot"></span><span>Online</span></span>
      <span class="header-clock" id="header-clock"></span>
    </div>`;
  const clock = document.getElementById('header-clock');
  const tick = () => { clock.textContent = new Date().toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit', second: '2-digit' }); };
  tick(); setInterval(tick, 1000);
  watchConnection();
}

// Indicador de conexão com o cbw-live (o servidor), atualizado a cada 5 s.
function watchConnection() {
  const badge = document.getElementById('conn-badge');
  const check = async () => {
    let ok = false;
    try { ok = (await fetch('/healthz', { cache: 'no-store' })).ok; } catch {}
    badge.className = `badge ${ok ? 'ok live' : 'bad'}`;
    badge.lastElementChild.textContent = ok ? 'Online' : 'Sem conexão';
  };
  check(); setInterval(check, 5000);
}

/** Escapa texto para uso em HTML. */
export function esc(v) {
  return String(v ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

export function flagURL(code) {
  return code ? `/flags/${encodeURIComponent(String(code).toLowerCase())}.svg` : '';
}

let toastWrap;
/** Aviso flutuante. kind: '' | 'ok' | 'bad' */
export function toast(msg, kind = '') {
  if (!toastWrap) {
    toastWrap = document.createElement('div');
    toastWrap.className = 'toast-wrap';
    toastWrap.setAttribute('role', 'status');
    document.body.append(toastWrap);
  }
  const t = document.createElement('div');
  t.className = `toast ${kind}`;
  t.textContent = msg;
  toastWrap.append(t);
  setTimeout(() => t.remove(), 3200);
}

export async function copyText(text) {
  try { await navigator.clipboard.writeText(text); toast('Link copiado', 'ok'); }
  catch { prompt('Copie o link:', text); }
}

/** "há 3 s", "há 2 min" */
export function ago(iso) {
  if (!iso) return '—';
  const s = Math.max(0, Math.round((Date.now() - Date.parse(iso)) / 1000));
  if (s < 60) return `há ${s} s`;
  if (s < 3600) return `há ${Math.round(s / 60)} min`;
  return `há ${Math.round(s / 3600)} h`;
}

/** Dados de exibição de um tapete a partir do /api/state. */
export function matView(m) {
  const f = m.fight || null;
  const live = m.live?.message && !m.live.message.idle && (!f || !m.live.message.id || m.live.message.id === f.id) ? m.live : null;
  const msg = live?.message;
  const side = (n, key) => ({
    name: msg?.[key]?.name || f?.[`fighter${n}DisplayName`] || f?.[`fighter${n}FullName`] || '',
    country: msg?.[key]?.country || f?.[`team${n}AlternateName`] || '',
    score: msg?.[key]?.score ?? 0,
    cautions: msg?.[key]?.cautions ?? 0,
  });
  return {
    hasFight: !!(f || msg),
    home: side(1, 'home'), away: side(2, 'away'),
    time: msg?.time || '',
    running: live?.clockRunning === true || (live?.clockRunning === undefined && live?.state === 'running' && m.liveAt && Date.now() - Date.parse(m.liveAt) < 2600),
    category: msg?.weightCategory || f?.weightCategoryShortName || '',
    round: msg?.round || f?.roundFriendlyName || '',
    matchNo: msg?.matchNo || f?.fightNumber || '',
    timing: m.timingAddr || '',
    liveRecent: m.liveAt && Date.now() - Date.parse(m.liveAt) < 15000,
    control: m.control || {},
  };
}
