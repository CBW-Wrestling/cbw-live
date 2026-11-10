// Modelos visuais do overlay: cada um é só marcação (HTML). O estilo fica em overlay.css (clássico)
// e themes.css (demais). Todos usam os mesmos ids, que o overlay.js preenche:
//   placar:      #bug  #logo #phase #category #period #pn #clock
//                f-/n-/c-/k-/s-/x-/p-{home|away}  (bandeira, nome, país, cartões, pontos, caixa lateral, painel)
//   apresentação: #intro  if-/ic-/in-{home|away}  #in-cat #in-phase #in-no #intro-band #intro-start #ie-name #ie-city
// O contêiner de cada atleta leva data-side="home|away" (recebe as classes adv / winner / loser).
// Para criar um modelo novo: adicione aqui, em themes.css e na lista de internal/live/settings.go.

const side = (s) => ({
  flag: `<img class="flag" id="f-${s}" alt="">`,
  name: `<span class="name" id="n-${s}"></span>`,
  code: `<span class="code" id="c-${s}"></span>`,
  cards: `<span class="cards" id="k-${s}"></span>`,
  score: `<span class="score" id="s-${s}"><b>0</b></span>`,
  box: `<span class="side-box" id="x-${s}"></span>`,
  panel: `<div class="panel" id="p-${s}"><div class="panel-in"><span class="p-name"></span><span class="p-status"></span></div></div>`,
});
const H = side('home'), A = side('away');
const LOGO = `<img class="logo" id="logo" src="/assets/logo.svg" alt="" onerror="this.remove()">`;
const CLOCK = `<span class="clock" id="clock">0:00</span>`;
const TXT = (...ids) => ids.map((i) => `<span id="${i}"></span>`).join('');

export const POSITIONS = [['top-start', 'Topo · início'], ['top-center', 'Topo · centro'], ['top-end', 'Topo · fim'], ['bottom-start', 'Base · início'], ['bottom-center', 'Base · centro'], ['bottom-end', 'Base · fim']];

// ---------------------------------------------------------------- placares
export const BOARDS = {
  classic: {
    label: 'Clássico CBW', desc: 'Dois cartões brancos com a barra de cor do atleta; fase e logo no topo.',
    html: () => `<section id="bug" class="bug out">
  <div class="phase">${LOGO}<span id="phase"></span></div>
  ${['home', 'away'].map((s) => { const x = side(s); return `<div class="row ${s}" data-side="${s}">${x.flag}
    <span class="who">${x.name}</span><span class="country">${x.cards}${x.code}</span><span class="bar"></span>${x.score}${x.box}${x.panel}</div>`; }).join('')}
  <div class="foot"><span id="category"></span><span id="period"></span>${CLOCK}</div>
</section>`,
  },

  broadcast: {
    label: 'Faixa azul', desc: 'Barra larga azul: logo e relógio à esquerda, atletas dos lados, fase e categoria numa tira acima.',
    html: () => `<section id="bug" class="sb sb-broadcast out">
  <div class="bc-top">${TXT('phase', 'category', 'period')}</div>
  <div class="bc-main">
    <div class="bc-badge"><span class="bc-logo">${LOGO}</span>${CLOCK}</div>
    <div class="bc-side home" data-side="home"><i class="bc-stripe"></i>${H.flag}${H.code}${H.name}${H.cards}${H.box}${H.score}${H.panel}</div>
    <div class="bc-side away" data-side="away"><i class="bc-stripe"></i>${A.flag}${A.code}${A.name}${A.cards}${A.box}${A.score}${A.panel}</div>
  </div>
</section>`,
  },

  stack: {
    label: 'Pilha ciano', desc: 'Compacto e vertical: uma linha por atleta e o tempo numa aba ciano embaixo.',
    html: () => `<section id="bug" class="sb sb-stack out">
  <div class="st-head">${TXT('phase', 'category')}</div>
  <div class="st-row home" data-side="home"><i class="st-stripe"></i><span class="st-team">${H.flag}${H.code}${H.name}${H.cards}</span>${H.box}${H.score}${H.panel}</div>
  <div class="st-row away" data-side="away"><i class="st-stripe"></i><span class="st-team">${A.flag}${A.code}${A.name}${A.cards}</span>${A.box}${A.score}${A.panel}</div>
  <div class="st-foot">${LOGO}<span id="period"></span>${CLOCK}</div>
</section>`,
  },

  compact: {
    label: 'Bloco duplo', desc: 'Bloco branco com período e tempo, siglas em preto e pontos em branco, empilhados.',
    html: () => `<section id="bug" class="sb sb-compact out">
  <div class="cp-top">${TXT('phase', 'category')}</div>
  <div class="cp-main">
    <div class="cp-time"><b id="pn"></b>${CLOCK}</div>
    <div class="cp-rows">
      <div class="cp-row home" data-side="home"><span class="cp-team"><i class="cp-stripe"></i>${H.flag}${H.code}${H.name}${H.cards}${H.box}</span>${H.score}${H.panel}</div>
      <div class="cp-row away" data-side="away"><span class="cp-team"><i class="cp-stripe"></i>${A.flag}${A.code}${A.name}${A.cards}${A.box}</span>${A.score}${A.panel}</div>
    </div>
  </div>
</section>`,
  },

  league: {
    label: 'Liga roxo', desc: 'Blocos nas cores dos atletas, pontos num miolo branco com a logo e o tempo pendurado embaixo.',
    html: () => `<section id="bug" class="sb sb-league out">
  <div class="lg-main">
    <div class="lg-side home" data-side="home">${H.flag}${H.code}${H.name}${H.cards}${H.box}${H.score}${H.panel}</div>
    <span class="lg-logo">${LOGO}</span>
    <div class="lg-side away" data-side="away">${A.flag}${A.code}${A.name}${A.cards}${A.box}${A.score}${A.panel}</div>
  </div>
  <div class="lg-clock">${CLOCK}</div>
  <div class="lg-info">${TXT('phase', 'category', 'period')}</div>
</section>`,
  },

  futy: {
    label: 'Faixa verde', desc: 'Uma linha só: logo, siglas em verde, bandeiras e pontos em branco, período e tempo à direita.',
    html: () => `<section id="bug" class="sb sb-futy out">
  <div class="fu-top">${TXT('phase', 'category')}</div>
  <div class="fu-main">
    <span class="fu-logo">${LOGO}</span>
    <div class="fu-side home" data-side="home"><span class="fu-team">${H.code}${H.name}${H.cards}${H.box}</span><span class="fu-wh">${H.flag}${H.score}</span>${H.panel}</div>
    <i class="fu-sep"></i>
    <div class="fu-side away" data-side="away"><span class="fu-team">${A.code}${A.name}${A.cards}${A.box}</span><span class="fu-wh">${A.flag}${A.score}</span>${A.panel}</div>
    <span class="fu-period" id="period"></span>${CLOCK}
  </div>
</section>`,
  },

  arena: {
    label: 'Arena', desc: 'Barra escura angulada com siglas grandes, linha na cor do atleta, tempo numa aba central e aba de destaque.',
    html: () => `<section id="bug" class="sb sb-arena out">
  <div class="ar-tab">${TXT('phase', 'category')}</div>
  <div class="ar-main">
    <div class="ar-side home" data-side="home"><span class="ar-id">${H.code}${H.name}</span>${H.cards}${H.flag}${H.score}${H.box}${H.panel}</div>
    <span class="ar-mid">${LOGO}</span>
    <div class="ar-side away" data-side="away"><span class="ar-id">${A.code}${A.name}</span>${A.cards}${A.flag}${A.score}${A.box}${A.panel}</div>
  </div>
  <div class="ar-clock">${CLOCK}<b id="pn"></b></div>
</section>`,
  },

  clean: {
    label: 'Cartela clara', desc: 'Cartão claro com bandeira, nome e pontos coloridos; rodapé cinza com categoria, período e tempo.',
    html: () => `<section id="bug" class="sb sb-clean out">
  <div class="cl-row home" data-side="home">${H.flag}${H.name}${H.code}${H.cards}${H.box}${H.score}${H.panel}</div>
  <div class="cl-row away" data-side="away">${A.flag}${A.name}${A.code}${A.cards}${A.box}${A.score}${A.panel}</div>
  <div class="cl-foot"><span class="cl-info">${TXT('category', 'phase')}</span><span id="period"></span>${CLOCK}</div>
</section>`,
  },

  duel: {
    label: 'Duelo', desc: 'Dois blocos nas cores dos atletas com pontos grandes e um miolo escuro com tempo e período.',
    html: () => `<section id="bug" class="sb sb-duel out">
  <div class="dl-top">${TXT('phase', 'category')}</div>
  <div class="dl-main">
    <div class="dl-side home" data-side="home">${H.flag}${H.code}${H.name}${H.cards}${H.box}${H.score}${H.panel}</div>
    <div class="dl-mid">${CLOCK}<span id="period"></span></div>
    <div class="dl-side away" data-side="away">${A.flag}${A.code}${A.name}${A.cards}${A.box}${A.score}${A.panel}</div>
  </div>
</section>`,
  },

  flo: {
    label: 'Escuro duplo', desc: 'Linhas escuras com pontos em caixas vermelha e azul; faixa de logo em cima e rodapé claro com luta, período e tempo.',
    html: () => `<section id="bug" class="sb sb-flo out">
  <div class="fl-head">${LOGO}<span id="phase"></span></div>
  <div class="fl-row home" data-side="home">${H.flag}${H.name}${H.code}${H.cards}${H.box}${H.score}${H.panel}</div>
  <div class="fl-row away" data-side="away">${A.flag}${A.name}${A.code}${A.cards}${A.box}${A.score}${A.panel}</div>
  <div class="fl-foot"><span id="bout"></span><span id="period"></span>${CLOCK}</div>
</section>`,
  },

  dual: {
    label: 'Barra dupla', desc: 'Barra larga: nome de cada lado, pontos em branco e o tempo no centro; faixa de categoria embaixo.',
    html: () => `<section id="bug" class="sb sb-dual out">
  <div class="du-main">
    <div class="du-side home" data-side="home">${H.flag}${H.code}${H.name}${H.cards}${H.box}${H.score}${H.panel}</div>
    <div class="du-mid">${CLOCK}<span id="period"></span></div>
    <div class="du-side away" data-side="away">${A.flag}${A.code}${A.name}${A.cards}${A.box}${A.score}${A.panel}</div>
  </div>
  <div class="du-sub">${TXT('bout', 'category', 'phase')}</div>
</section>`,
  },

  judo: {
    label: 'Tabuleiro', desc: 'Faixa de categoria, evento e tempo em cima; logo grande à esquerda e uma linha por atleta nas cores dos cantos.',
    html: () => `<section id="bug" class="sb sb-judo out">
  <div class="jd-head"><span id="category"></span><span id="phase"></span><span id="period"></span>${CLOCK}</div>
  <div class="jd-body">
    <span class="jd-logo">${LOGO}</span>
    <div class="jd-rows">
      <div class="jd-row home" data-side="home">${H.flag}${H.code}${H.name}${H.cards}${H.box}${H.score}${H.panel}</div>
      <div class="jd-row away" data-side="away">${A.flag}${A.code}${A.name}${A.cards}${A.box}${A.score}${A.panel}</div>
    </div>
  </div>
</section>`,
  },

  uww: {
    label: 'UWW oficial', desc: 'Barra preta com nome e bandeira de cada lado, logo em círculo no centro, linha na cor do canto e relógio em aba embaixo.',
    html: () => `<section id="bug" class="sb sb-uww out">
  <div class="uw-top">${TXT('phase', 'category')}</div>
  <div class="uw-main">
    <div class="uw-side home" data-side="home">${H.name}${H.flag}${H.cards}${H.box}${H.score}${H.panel}</div>
    <span class="uw-mid"><span>${LOGO}</span></span>
    <div class="uw-side away" data-side="away">${A.name}${A.flag}${A.cards}${A.box}${A.score}${A.panel}</div>
  </div>
  <div class="uw-clock">${CLOCK}</div>
</section>`,
  },
};

// ---------------------------------------------------------------- apresentações
const fighter = (s, cls) => `<div class="${cls} ${s}"><img class="flag" id="if-${s}" alt=""><span class="who"><span class="code" id="ic-${s}"></span><span class="name" id="in-${s}"></span></span></div>`;
const EVENT = `<div class="event"><span id="ie-name"></span><span id="ie-city"></span></div>`;
const START = `<div class="start" id="intro-start"></div>`;

export const INTROS = {
  lower: {
    label: 'Tarja inferior', pos: 'bottom-center', desc: 'Atletas empilhados numa tarja escura, faixa da categoria e evento embaixo.',
    html: () => `<section id="intro" class="intro out">
  <div class="intro-start" id="intro-start"></div>
  <div class="intro-rows">
    <div class="intro-row home"><img class="flag" id="if-home" alt=""><span class="country" id="ic-home"></span><span class="name" id="in-home"></span></div>
    <div class="intro-row away"><img class="flag" id="if-away" alt=""><span class="country" id="ic-away"></span><span class="name" id="in-away"></span></div>
  </div>
  <div class="intro-band" id="intro-band"></div>
  <div class="intro-event"><span id="ie-name"></span><span id="ie-city"></span></div>
</section>`,
  },
  versus: {
    label: 'Versus', pos: 'bottom-center', desc: 'Faixa larga embaixo: um atleta de cada lado, "VS" no meio; as metades entram pelas laterais.',
    html: () => `<section id="intro" class="sbi sbi-versus out">
  ${START}
  <div class="vs-bar">${fighter('home', 'vs-side')}<div class="vs-mid"><b>VS</b><span id="in-no"></span></div>${fighter('away', 'vs-side')}</div>
  <div class="vs-info"><span id="in-cat"></span><span id="in-phase"></span></div>
  ${EVENT}
  <i id="intro-band" hidden></i>
</section>`,
  },
  card: {
    label: 'Cartão central', pos: 'middle-center', desc: 'Cartão no centro da tela: evento no topo, atletas em destaque e categoria no rodapé.',
    html: () => `<section id="intro" class="sbi sbi-card out">
  <div class="cd-head"><span id="ie-name"></span><span id="ie-city"></span></div>
  <div class="cd-body">${fighter('home', 'cd-fighter')}<div class="cd-vs"><b>VS</b></div>${fighter('away', 'cd-fighter')}</div>
  <div class="cd-foot"><span id="in-phase"></span><span id="in-cat"></span><span id="in-no"></span></div>
  ${START}
  <i id="intro-band" hidden></i>
</section>`,
  },
  topbar: {
    label: 'Barra no topo', pos: 'top-center', desc: 'Desce do topo no mesmo estilo dos placares centralizados, com a categoria numa aba embaixo.',
    html: () => `<section id="intro" class="sbi sbi-topbar out">
  <div class="tb-main">${fighter('home', 'tb-side')}<div class="tb-mid"><span id="in-phase"></span><span id="in-no"></span></div>${fighter('away', 'tb-side')}</div>
  <div class="tb-info"><span id="in-cat"></span>${TXT('ie-name', 'ie-city')}</div>
  ${START}
  <i id="intro-band" hidden></i>
</section>`,
  },
  arena: {
    label: 'Arena', pos: 'bottom-center', desc: 'Placas escuras anguladas com siglas grandes e linha na cor do atleta; aba amarela com a categoria.',
    html: () => `<section id="intro" class="sbi sbi-arena out">
  <div class="ai-tab"><span id="in-phase"></span><span id="in-cat"></span></div>
  <div class="ai-main">${fighter('home', 'ai-side')}<div class="ai-mid"><b>VS</b></div>${fighter('away', 'ai-side')}</div>
  <div class="ai-foot"><span id="ie-name"></span><span id="ie-city"></span><span id="in-no"></span></div>
  ${START}
  <i id="intro-band" hidden></i>
</section>`,
  },
  clean: {
    label: 'Cartela clara', pos: 'bottom-start', desc: 'Cartão claro compacto com os dois atletas e rodapé cinza com categoria e fase.',
    html: () => `<section id="intro" class="sbi sbi-clean out">
  <div class="cn-top"><span id="ie-name"></span><span id="ie-city"></span></div>
  ${fighter('home', 'cn-row')}${fighter('away', 'cn-row')}
  <div class="cn-foot"><span id="in-cat"></span><span id="in-phase"></span><span id="in-no"></span></div>
  ${START}
  <i id="intro-band" hidden></i>
</section>`,
  },
  split: {
    label: 'Tela dividida', pos: 'middle-center', desc: 'Grande destaque: duas faixas cruzam a tela, uma de cada cor, com os nomes em letras enormes.',
    html: () => `<section id="intro" class="sbi sbi-split out">
  <div class="sp-bands">${fighter('home', 'sp-side')}<div class="sp-mid"><b>VS</b></div>${fighter('away', 'sp-side')}</div>
  <div class="sp-info"><span id="in-phase"></span><span id="in-cat"></span><span id="in-no"></span></div>
  <div class="event"><span id="ie-name"></span><span id="ie-city"></span></div>
  ${START}
  <i id="intro-band" hidden></i>
</section>`,
  },
  ticker: {
    label: 'Faixa fina', pos: 'bottom-center', desc: 'Faixa baixa de uma linha só: fase, atletas frente a frente, categoria e evento.',
    html: () => `<section id="intro" class="sbi sbi-ticker out">
  <div class="tk-main"><span class="tk-phase" id="in-phase"></span>${fighter('home', 'tk-side')}<span class="tk-vs">VS</span>${fighter('away', 'tk-side')}<span class="tk-cat" id="in-cat"></span></div>
  <div class="tk-event"><span id="ie-name"></span><span id="ie-city"></span></div>
  ${START}
  <i id="intro-band" hidden></i>
</section>`,
  },
  uww: {
    label: 'UWW oficial', pos: 'top-center', desc: 'Barra preta larga com nomes e bandeiras, logo em círculo no centro e abas de categoria e luta; abre do centro para as pontas.',
    html: () => `<section id="intro" class="sbi sbi-uww out">
  <div class="uwi-top"><span id="in-phase"></span><span id="in-cat"></span></div>
  <div class="uwi-main">${fighter('home', 'uwi-side')}<span class="uwi-mid"><span><img class="logo" src="/assets/logo.svg" alt="" onerror="this.remove()"></span></span>${fighter('away', 'uwi-side')}</div>
  <div class="uwi-tab"><span id="in-no"></span></div>
  ${EVENT}
  ${START}
  <i id="intro-band" hidden></i>
</section>`,
  },
};

export const ANIMS = [
  ['auto', 'Do modelo', 'Cada modelo usa a sua entrada'],
  ['fade', 'Suave', 'Aparece e some'],
  ['slide', 'Deslizar', 'Entra pelo lado da posição escolhida'],
  ['drop', 'Da borda', 'Entra pela borda da tela, em cima ou embaixo'],
  ['zoom', 'Zoom', 'Cresce do centro'],
  ['wipe', 'Revelar', 'Abre como uma cortina'],
];
