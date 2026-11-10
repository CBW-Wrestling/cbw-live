// Cenários fictícios para a galeria de estados (/preview.html) e para
// overlay.html?mock=<cenário>. Não dependem do Arena nem do UWW Timing.
// Formatos iguais aos reais: snapshot do cbwlive + mensagem ao vivo do Timing.

const FIGHT = {
  id: 'mock-fight',
  fighter1FullName: 'Hassan Sabzali RAHIMI', fighter1DisplayName: 'RAHIMI H.', team1AlternateName: 'IRI',
  fighter2FullName: 'Victor LEBEDEV', fighter2DisplayName: 'LEBEDEV V.', team2AlternateName: 'RUS',
  roundFriendlyName: 'Qualif.', fightNumber: 12, roundDuration: 180, roundsNumber: 2,
  activityTime: 30, injuryTime: 240,
  sportName: 'Freestyle', audienceName: 'Seniors', weightCategoryName: '57 kg', weightCategoryShortName: 'Seniors FS 57',
};
const NATIONAL = {
  ...FIGHT,
  fighter1FullName: 'João Pedro da SILVA', fighter1DisplayName: 'SILVA J.', team1AlternateName: 'PB',
  fighter2FullName: 'Lucas Henrique OLIVEIRA', fighter2DisplayName: 'OLIVEIRA L.', team2AlternateName: 'SP',
  roundFriendlyName: '1/2', weightCategoryShortName: 'Cadets GR 60', weightCategoryName: '60 kg', sportName: 'Greco-Roman', audienceName: 'Cadets',
};
const LONG = {
  ...FIGHT,
  fighter1FullName: 'Abdulrashid Bulachevich SADULAEV', fighter1DisplayName: 'SADULAEV ABDULRASHID B.',
  fighter2FullName: 'Kyle Frederick SNYDER-WILLIAMSON', fighter2DisplayName: 'SNYDER-WILLIAMSON K.', team2AlternateName: 'USA',
};

const live = (fight, o = {}) => ({
  state: o.state ?? 'running',
  matName: 'A',
  message: {
    id: fight.id, idle: false,
    weightCategory: fight.weightCategoryShortName, round: fight.roundFriendlyName, matchNo: fight.fightNumber,
    time: o.time ?? '4:12', advantage: o.advantage ?? '',
    home: { name: fight.fighter1DisplayName, country: fight.team1AlternateName, score: o.s1 ?? 2, cautions: o.k1 ?? 0, active: true },
    away: { name: fight.fighter2DisplayName, country: fight.team2AlternateName, score: o.s2 ?? 0, cautions: o.k2 ?? 0, active: true },
  },
});

const ctl = (graphic, o = {}) => ({ [graphic]: { graphic, visible: true, startedAt: Date.now() - (o.elapsed ?? 0) * 1000, ...o } });

// Cada cenário: { title, group, fight, live, control, running?, steps? }
// control pode ser função (recalculada a cada ciclo, para os cronômetros recomeçarem).
// running: simula o cronômetro correndo (mensagens chegando) para o modo recolhido automático.
// steps: [ [ms, (s) => mudanças ] ] — repetido em ciclo para mostrar animações.
export const SCENARIOS = {
  'pre-luta':        { group: 'Placar', title: 'Antes da luta (expandido)', fight: FIGHT, live: live(FIGHT, { state: 'starting', time: '6:00', s1: 0, s2: 0 }) },
  'correndo':        { group: 'Placar', title: 'Luta correndo (recolhido automático)', fight: FIGHT, live: live(FIGHT), running: true },
  'parado':          { group: 'Placar', title: 'Luta parada (expandido)', fight: FIGHT, live: live(FIGHT, { s1: 3, s2: 1 }) },
  'empate':          { group: 'Placar', title: 'Empate — último a pontuar (barra branca)', fight: FIGHT, live: live(FIGHT, { s1: 2, s2: 2, advantage: 'away' }) },
  'faltas':          { group: 'Placar', title: 'Faltas (cartões amarelos)', fight: FIGHT, live: live(FIGHT, { s1: 1, s2: 4, k1: 2, k2: 1 }) },
  'final':           { group: 'Placar', title: '10 segundos finais', fight: FIGHT, live: live(FIGHT, { time: '0:07', s1: 5, s2: 4 }) },
  'pontuando':       { group: 'Placar', title: 'Pontuação (animação)', fight: FIGHT, live: live(FIGHT, { s1: 0, s2: 0 }),
    steps: [[1200, (s) => { s.live.message.home.score = 2; }], [2600, (s) => { s.live.message.away.score = 4; }], [4200, (s) => { s.live.message.home.score = 0; s.live.message.away.score = 0; }]] },
  'recolhe':         { group: 'Placar', title: 'Expandido ↔ recolhido (animação)', fight: FIGHT, live: live(FIGHT),
    steps: [[1500, (s) => { s.control = { layout: { graphic: 'layout', status: 'collapsed' } }; }], [3500, (s) => { s.control = {}; }]] },

  'challenge-requested': { group: 'Challenge', title: 'Challenge requested', fight: FIGHT, live: live(FIGHT), control: () => ctl('challenge', { side: 'away', status: 'requested' }) },
  'challenge-analise':   { group: 'Challenge', title: 'Challenge (em análise)', fight: FIGHT, live: live(FIGHT), control: () => ctl('challenge', { side: 'away', status: 'challenge' }) },
  'challenge-won':       { group: 'Challenge', title: 'Challenge won', fight: FIGHT, live: live(FIGHT), control: () => ctl('challenge', { side: 'home', status: 'won' }) },
  'challenge-lost':      { group: 'Challenge', title: 'Challenge lost', fight: FIGHT, live: live(FIGHT), control: () => ctl('challenge', { side: 'home', status: 'lost' }) },
  'challenge-canceled':  { group: 'Challenge', title: 'Challenge canceled', fight: FIGHT, live: live(FIGHT), control: () => ctl('challenge', { side: 'away', status: 'canceled' }) },
  'challenge-fluxo':     { group: 'Challenge', title: 'Fluxo completo (animação)', fight: FIGHT, live: live(FIGHT), control: {},
    steps: [
      [800, (s) => { s.control = ctl('challenge', { side: 'away', status: 'requested' }); }],
      [2600, (s) => { s.control = ctl('challenge', { side: 'away', status: 'challenge' }); }],
      [4400, (s) => { s.control = ctl('challenge', { side: 'away', status: 'won' }); }],
      [6400, (s) => { s.control = {}; }],
    ] },

  'passividade':     { group: 'Tempo', title: 'Passividade (30 s)', fight: FIGHT, live: live(FIGHT), control: () => ctl('passivity', { side: 'home', elapsed: 13 }) },
  'passividade-fim': { group: 'Tempo', title: 'Passividade — últimos segundos', fight: FIGHT, live: live(FIGHT), control: () => ctl('passivity', { side: 'home', elapsed: 26 }) },
  'sangue':          { group: 'Tempo', title: 'Atendimento — sangue', fight: FIGHT, live: live(FIGHT), control: () => ctl('injury', { side: 'away', kind: 'blood', elapsed: 35 }) },
  'lesao':           { group: 'Tempo', title: 'Atendimento — lesão (últimos 30 s)', fight: FIGHT, live: live(FIGHT), control: () => ctl('injury', { side: 'home', kind: 'injury', elapsed: 215 }) },

  'vitoria':         { group: 'Resultado', title: 'Vitória (Victory by fall)', fight: FIGHT, live: live(FIGHT, { s1: 6, s2: 2, time: '2:31' }), control: () => ctl('victory', { side: 'home', victoryType: 'VICTORY BY FALL' }) },
  'vitoria-pontos':  { group: 'Resultado', title: 'Vitória por pontos (azul)', fight: FIGHT, live: live(FIGHT, { s1: 3, s2: 5, time: '0:00' }), control: () => ctl('victory', { side: 'away', victoryType: 'VICTORY BY POINTS' }) },

  'apresentacao':    { group: 'Apresentação', title: 'Apresentação da luta (match ready)', fight: FIGHT, live: null, control: () => ctl('intro') },
  'apresentacao-nac':{ group: 'Apresentação', title: 'Apresentação — evento nacional (estados)', fight: NATIONAL, live: null, control: () => ctl('intro') },

  'nacional':        { group: 'Variações', title: 'Evento nacional (bandeiras de estado)', fight: NATIONAL, live: live(NATIONAL, { s1: 4, s2: 1 }) },
  'nomes-longos':    { group: 'Variações', title: 'Nomes longos', fight: LONG, live: live(LONG, { s1: 10, s2: 8 }) },
};

export function buildState(name) {
  const sc = SCENARIOS[name];
  if (!sc) return null;
  return {
    sc,
    snapshot: {
      eventId: 'mock', eventName: 'Campeonato Brasileiro de Wrestling', eventCity: 'Campina Grande', matName: 'A',
      colors: { athlete1: '#ba0000', athlete2: '#0000aa' },
      fight: sc.fight, control: typeof sc.control === 'function' ? sc.control() : structuredClone(sc.control || {}), serverTime: Date.now(),
    },
    live: sc.live ? structuredClone(sc.live) : null,
  };
}
