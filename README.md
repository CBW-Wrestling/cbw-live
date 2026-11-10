# cbw-live

Gráficos de transmissão ao vivo da **CBW (Confederação Brasileira de Wrestling)** integrados ao **UWW Arena** e ao **UWW Timing**: placar, cronômetro, apresentação da luta, challenge, atendimento, passividade e vitória, prontos para OBS, vMix ou qualquer navegador. É uma alternativa própria ao LGTV da UWW.

```
                       ┌──────────────────────────── Mac / servidor ───────────────────────────┐
 UWW Timing (Windows) ──┼─▶ Arena (Docker :8080) ──webhook──▶ cbw-live (:8095) ──▶ overlays    │
   pontos, tempo,       │        │  REST/OAuth  ◀──────────────────┘     │          (OBS, vMix, │
   challenge, vitória   │        └─ Mercure (SSE) ◀── publica/assina ────┘           navegador) │
                       └───────────────────────────────────────────────────────────────────────┘
```

- **Placar ao vivo** no estilo das transmissões da UWW: bandeira, nome, país, pontos, faltas, tempo, fase e categoria.
- **Expandido e recolhido:** o nome sai da tela enquanto o cronômetro corre e volta nas paradas.
- **Gráficos automáticos** a partir dos dados do Timing e do Arena: apresentação da luta, challenge (won/lost) e vitória com o tipo oficial.
- **Página de controle** para o operador acionar o que o Timing não envia: atendimento (sangue/lesão), passividade e challenge pedido/em análise.
- **Eventos nacionais e internacionais:** bandeiras dos 27 estados (as mesmas do UWW Timing) e de países (servidas pelo próprio Arena).
- **Fundo transparente ou chroma key** (verde/azul).
- **Leve e rápido:** um único executável em Go (~10 MB, sem runtime), republica cada atualização em ~3 ms e distribui pelo Mercure que já roda no Arena.

---

## Sumário
- [Como funciona](#como-funciona)
- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Uso](#uso)
- [Telas](#telas)
- [Configurações](#configurações)
- [Uso no OBS](#uso-no-obs)
- [Uso no vMix](#uso-no-vmix)
- [Uso na web](#uso-na-web)
- [Refresh (atualizar os overlays)](#refresh-atualizar-os-overlays)
- [O que é automático e o que é manual](#o-que-é-automático-e-o-que-é-manual)
- [Rede e endereços](#rede-e-endereços)
- [Solução de problemas](#solução-de-problemas)
- [Desenvolvimento](#desenvolvimento)
- [Segurança](#segurança)
- [Limitações conhecidas e próximos passos](#limitações-conhecidas-e-próximos-passos)

---

## Como funciona

| Fonte | O que fornece | Como o cbw-live recebe |
|---|---|---|
| **UWW Timing** (mesa) | Pontos, faltas, tempo, quem pontuou por último (`advantage`), estado da luta (`starting`/`running`), vencedor e tipo de vitória | Assina o tópico Mercure `http://arena.uww.org/live/{evento}/{tapete}`, onde o Timing publica |
| **UWW Arena** (REST) | Eventos, tapetes, luta atual, nomes completos, bandeiras, categoria, fase, durações, tipos de vitória, detalhes do challenge | API `/api/json/scoreboard/*` com OAuth (`grant_type=https://arena.uww.io/grants/api_key`) |
| **UWW Arena** (webhooks) | Avisos de mudança (`Fight`, `RefereeChallenge`, `RoundTechnicalPoint`, …) | `POST /webhook/arena` |
| **Operador** | Atendimento, passividade, challenge pedido/em análise, ajustes | Página de controle |

O cbw-live mantém o estado de cada tapete em memória e publica no Mercure, no tópico `cbw/live/{evento}/{tapete}`. Os overlays assinam pelo proxy `/hub` do próprio serviço, na mesma origem e com o JWT guardado no servidor.

O placar não consulta o Arena durante a luta. Uma ressincronização de segurança roda a cada 15 s.

## Requisitos

- **UWW Arena** rodando em Docker (instalação oficial `docker-compose`), acessível em `http://<ip>:8080`.
- **Go 1.22+**, só para compilar. No macOS: `brew install go`.
- **UWW Timing** (Windows) na mesma rede, para operar as lutas.
- OBS Studio, vMix ou um navegador moderno para exibir os overlays.

## Instalação

### 1. Credenciais no Arena
No Arena (`http://<ip>:8080`), logado como administrador:

1. **Settings → Users → Add a new user**: usuário `cbwlive`, papel **SCOREBOARD**. Depois clique em **(Re)generate API key**.
2. **Settings → Apps → Add a new app**: nome `CBW Live`, *Redirect uris* `http://localhost:8080` (o campo é obrigatório, mas não é usado), *Allowed grant types* `https://arena.uww.io/grants/api_key`.

> Não reaproveite o app **Scoreboard**, que é o do UWW Timing. Com credenciais separadas, trocar uma não derruba a outra.

### 2. Configuração (`.env`)
```bash
git clone https://github.com/CBW-Wrestling/cbw-live.git
cd cbw-live
./scripts/setup-env.sh            # ou: ./scripts/setup-env.sh /caminho/do/docker-compose/do/arena
```
O script pede a URL do Arena, o App ID e o Secret do app **CBW Live** e a API Key do usuário **cbwlive**. A chave do Mercure ele lê sozinho do container `caddy` do Arena. O `.env` é criado com permissão 600 e nunca vai para o git. Modelo: [`.env.example`](.env.example).

### 3. Compilar e rodar
```bash
go build -o bin/cbwlive ./cmd/cbwlive
./bin/cbwlive                     # usa ./.env; ./bin/cbwlive -debug para log detalhado
```
Abra **http://localhost:8095**.

Executável para o notebook Windows do evento:
```bash
GOOS=windows GOARCH=amd64 go build -o bin/cbwlive.exe ./cmd/cbwlive
```

### 4. Webhook no Arena
**Settings → Webhooks → Add a new Webhook**:
- **Url:** `http://host.docker.internal:8095/webhook/arena` (Arena em Docker na mesma máquina) ou `http://<ip-do-cbw-live>:8095/webhook/arena`
- **Token:** o `WEBHOOK_TOKEN` do `.env` (opcional por enquanto; veja [Segurança](#segurança))
- **Status:** Enabled
- **Eventos (insert/update/delete):** Fight, FightMat, Fighter, RefereeChallenge, Round, RoundCaution, RoundTechnicalPoint, SportEventMat

### 5. UWW Timing
No Timing (Windows), configure o Arena com o **IP da máquina do Arena** (não `localhost`) e as credenciais do app/usuário **Scoreboard**. Selecione o evento e o tapete normalmente.

## Uso

1. Suba o Arena e o cbw-live.
2. Abra **http://<ip>:8095**: a lista mostra todos os eventos e tapetes, com os links de **overlay** e **controle**.
3. Coloque o link de **overlay** de cada tapete no OBS/vMix ([abaixo](#uso-no-obs)).
4. Abra a página de **controle** do tapete para o operador.
5. Opere as lutas no UWW Timing. O placar, a apresentação, o challenge e a vitória seguem sozinhos.

Novos eventos e tapetes criados no Arena aparecem automaticamente: a cada webhook ou, no máximo, em 15 s.

## Telas

| Tela | URL | Para quê |
|---|---|---|
| **Lista de tapetes** | `/` | Todos os eventos e tapetes, status do placar ao vivo, links de overlay e controle |
| **Overlay** | `/overlay.html?event=<id>&mat=<tapete>` | O gráfico de transmissão (1920×1080). É a fonte do OBS/vMix |
| **Controle do operador** | `/control.html?event=<id>&mat=<tapete>` | Botões dos gráficos manuais + prévia ao vivo do overlay |
| **Configurações** | `/settings.html` | Preferências globais (valem para todos os tapetes) |
| **Galeria de estados** | `/preview.html` | Todos os estados do overlay com dados fictícios, para revisar o visual sem Arena nem Timing |

### Overlay
- **Placar** (canto superior esquerdo): logo CBW, fase, bandeira, nome, cartões de falta, país, pontos, categoria (`Seniors | FS | 57`), período e cronômetro.
  - Contorno branco no quadro de pontos de **quem pontuou por último** quando o placar está empatado.
  - Cronômetro **vermelho pulsando nos 10 segundos finais**.
  - **Expandido/recolhido:** recolhido mostra só bandeira, país, pontos e tempo.
- **Painel lateral:** challenge (requested / em análise / won / lost) e vencedor ("Winner · VICTORY BY FALL").
- **Caixa lateral:** passividade (contagem de 30 s) ou atendimento (sangue/lesão, 4:00, piscando nos últimos 30 s).
- **Apresentação da luta** (tarja inferior): bandeiras, países, nomes completos, "Freestyle - Seniors - 57 kg  Qualification" e "EVENTO | Cidade". O placar fica oculto enquanto ela está no ar.
- Opções de URL: `&debug=1` mostra os dados brutos; `&bg=green|blue|transparent` força o fundo daquele overlay.

### Controle do operador
- **Placar neste tapete:** Padrão (segue as configurações), Expandido ou Recolhido.
- **Apresentação:** Mostrar / Esconder (atalho **I**).
- **Challenge:** Vermelho pediu / Azul pediu → Em análise → Ganhou / Perdeu → Esconder.
- **Atendimento:** Vermelho/Azul × Sangue/Lesão, e Encerrar.
- **Passividade:** Vermelho / Azul, e Encerrar.
- **Vitória:** tipo de vitória (lista oficial do Arena) + Vermelho/Azul venceu.
- **Limpar todos os gráficos** (atalho **Esc**).
- Cada seção mostra o que está no ar. Vários operadores ou abas ficam sincronizados.
- Ao carregar uma nova luta no Timing, os gráficos da luta anterior saem sozinhos.

### Galeria de estados
Placar (antes da luta, correndo, parado, empate, faltas, 10 s finais, pontuação, expandir/recolher), challenge (cada status e fluxo completo), tempo (passividade, sangue, lesão), resultado, apresentação (internacional/nacional) e variações (estados, nomes longos). Cada cena tem **↻ animar** e **tela cheia**. Uma cena isolada abre em `/overlay.html?mock=<cena>`, e é possível escolher o fundo (tapete, escuro, claro, xadrez).

## Configurações

### Tela de configurações (`/settings.html`)
Salvas em `data/settings.json` e aplicadas na hora em todos os overlays abertos, sem recarregar.

| Opção | Valores | Padrão |
|---|---|---|
| Modo recolhido | Automático · Nunca recolher · Sempre recolhido | Automático |
| Expandir após um ponto | 0–60 s (só no automático) | 6 s |
| Fundo do overlay | Transparente · Verde chroma (#00FF00) · Azul chroma (#0000FF) | Transparente |
| Apresentação automática ao carregar a luta | liga/desliga | ligado |
| Mostrar nome do evento e cidade | liga/desliga | ligado |
| Mostrar "Início em" | liga/desliga (usa o horário previsto do Arena; confira o fuso do evento) | desligado |

**Modo automático do placar:** expandido antes da luta, nas paradas, logo após um ponto e com challenge/atendimento/vitória no ar; recolhido enquanto o cronômetro corre. O Timing não avisa pausa, então o cbw-live considera a luta parada depois de ~2,6 s sem mensagem.

### Variáveis de ambiente (`.env`)
| Variável | Descrição | Padrão |
|---|---|---|
| `ARENA_URL` | Endereço do Arena | `http://localhost:8080` |
| `ARENA_CLIENT_ID` / `ARENA_CLIENT_SECRET` | App ID e Secret (Settings → Apps) | — |
| `ARENA_API_KEY` | API Key do usuário de placar (Settings → Users) | — |
| `ARENA_EVENT_ID` | Acompanhar só um evento (vazio = todos) | vazio |
| `MERCURE_HUB_URL` | Hub Mercure do Arena | `http://localhost:8080/.well-known/mercure` |
| `MERCURE_JWT_KEY` | `MERCURE_PUBLISHER_JWT_KEY` do container `caddy` | — |
| `LISTEN_ADDR` | Porta do cbw-live | `:8095` |
| `DATA_DIR` | Logs e configurações | `data` |
| `ASSETS_DIR` | Logo e bandeiras | `assets` |
| `WEBHOOK_TOKEN` / `WEBHOOK_TOKEN_ENFORCE` | Token do webhook / rejeitar se não conferir | gerado / `false` |
| `RESYNC_SECONDS` | Ressincronização de segurança com o Arena (0 = desliga) | `15` |

### Logo e bandeiras
- **Logo:** `assets/logo.svg`, exibida no cabeçalho do placar. Sem o arquivo, o placar se ajusta sozinho.
- **Bandeiras de estado:** `assets/flags/<sigla>.svg` (`pb.svg`, `sp.svg`…). O nome do arquivo é o código da equipe no Arena (`teamAlternateName`). Essas bandeiras têm prioridade sobre as de país, que vêm do próprio Arena (`/build/images/flags/custom/`). Todas aparecem em círculo.
- Arquivos em `assets/` podem ser trocados com o serviço rodando. Basta recarregar o overlay.

## Uso no OBS
1. **Fontes → + → Navegador** (Browser Source).
2. **URL:** `http://<ip>:8095/overlay.html?event=<id>&mat=<tapete>` (copie da lista de tapetes).
3. **Largura 1920**, **Altura 1080**.
4. Deixe desmarcados "Desligar fonte quando não visível" e "Atualizar navegador quando a cena ficar ativa". Assim o overlay fica sempre conectado e as animações não reiniciam.
5. **CSS personalizado:** apague o CSS padrão do OBS (ou deixe `body { background-color: rgba(0,0,0,0); }`).
6. Com fundo **transparente** (padrão), não precisa de chroma. Se usar **verde/azul**, adicione o filtro **Chroma Key** com a cor correspondente.

Uma fonte por tapete. Para vários tapetes na mesma cena, duplique a fonte e troque o `mat=`.

## Uso no vMix
1. **Add Input → Web Browser**.
2. **URL** do overlay e **resolução 1920×1080**.
3. vMix suporta transparência na entrada Web Browser: use o fundo **transparente** e coloque o input numa camada (**Overlay 1–4**) sobre a câmera.
4. Alternativa para mesas/switchers que recortam por cor: configure **Fundo do overlay → Verde chroma** e use **Colour Key** (verde #00FF00) no input.
5. Para atualizar: clique com o botão direito no input → **Web Browser → Reload**.

## Uso na web
- Qualquer navegador abre o overlay. Ele **se ajusta ao tamanho da janela**, então o quadro inteiro cabe numa janela menor (notebook, telão, TV).
- **Telão ou TV na arena:** abra o overlay em tela cheia (F11) num navegador ligado à TV, com fundo transparente ou com o fundo de chroma, conforme o caso.
- **Celular ou tablet do operador:** a página de controle funciona em telas pequenas.
- Todas as telas funcionam por IP na rede local, sem internet.

## Refresh (atualizar os overlays)

| Situação | Precisa recarregar? |
|---|---|
| Pontos, tempo, nova luta, gráficos, mudanças de configuração | **Não**: chega ao vivo |
| cbw-live reiniciado | **Não**: os overlays reconectam sozinhos e recuperam o estado completo |
| Troca de `assets/logo.svg` ou de bandeiras | **Sim** |
| Atualização de versão do cbw-live (HTML/CSS/JS novos) | **Sim** |

Como recarregar:
- **OBS:** propriedades da fonte → **Atualizar cache da página atual** (Refresh cache of current page).
- **vMix:** botão direito no input → **Web Browser → Reload**.
- **Navegador:** **Cmd+Shift+R** (Mac) / **Ctrl+Shift+R** (Windows), recarregamento sem cache.

## O que é automático e o que é manual

| Gráfico | Fonte | Observação |
|---|---|---|
| Placar, tempo, faltas, último a pontuar | Timing (ao vivo) | Automático |
| Apresentação da luta | Timing (`starting` → `running`) | Automático (configurável); entra ao carregar a luta e sai no start |
| Vitória | Timing (`message.detail`) | Automático, com o nome oficial do tipo de vitória |
| Challenge **won/lost** | Webhook `RefereeChallenge` + Arena | Automático; fica 5 s no ar |
| Challenge **pedido / em análise** | — | **Manual:** o Arena só registra o challenge já decidido |
| Atendimento (sangue/lesão) | — | **Manual:** o Timing não publica |
| Passividade | — | **Manual:** o Timing não publica |

Os botões da página de controle sempre podem corrigir ou forçar qualquer gráfico.

## Rede e endereços
| Serviço | Porta | Quem acessa |
|---|---|---|
| Arena | 8080 | UWW Timing, administração, cbw-live |
| cbw-live | 8095 | OBS, vMix, navegadores, operador; webhook do Arena |

- O UWW Timing e as máquinas de produção precisam do **IP da máquina** (ex.: `http://192.168.0.7:8095`). `localhost` só funciona na própria máquina.
- No evento, **fixe o IP** da máquina do Arena/cbw-live no roteador (reserva de DHCP).
- Prefira **cabo** na máquina do Timing: com Wi-Fi ruim, o Timing já foi visto publicando a cada 2 s em vez de a cada 1 s.

## Solução de problemas
| Sintoma | Causa provável / solução |
|---|---|
| Log `OUTRO PROGRAMA responde nesta porta via IPv4` | Outro app usa a porta (já aconteceu com o REAPER na 8090). Troque `LISTEN_ADDR` |
| Overlay vazio | Nenhuma luta aberta no tapete, ou `event`/`mat` errados na URL. Confira na lista de tapetes |
| Webhooks não chegam | Confira o webhook em Settings → Webhooks (Enabled) e a URL `host.docker.internal:8095`. Os recebidos ficam em `data/webhooks.jsonl` |
| `invalid_client` / `invalid_grant` no log | App ID/Secret ou API Key errados, ou o app não usa o grant `.../grants/api_key` |
| Placar atrasado ~1 s | Normal: o Timing publica uma vez por segundo, na troca do segundo. O cbw-live adiciona ~3 ms |
| Apresentação não aparece | Desligada em Configurações, ou a luta já tinha sido apresentada (ela aparece uma vez por luta) |
| Gráfico velho na tela após atualização | Faça o [refresh](#refresh-atualizar-os-overlays) |

### Logs de descoberta (`data/`)
| Arquivo | Conteúdo |
|---|---|
| `webhooks.jsonl` | Cada webhook recebido do Arena |
| `uww-live.jsonl` | Cada mensagem ao vivo do UWW Timing |
| `mercure-all.jsonl` | Tudo o que passou pelo hub Mercure |
| `settings.json` | Configurações salvas |

## Desenvolvimento
```bash
go test -race ./...               # testes (inclui mensagens reais do Timing em internal/live/testdata)
go vet ./...
go run ./cmd/cbwlive -debug       # rodar sem compilar
./bin/arenaget scoreboard/sport-event   # consultar a API do Arena com as credenciais do .env
```
As páginas em `web/` são embutidas no executável (`go:embed`), então é preciso recompilar após editar.

```
cmd/cbwlive/        serviço (HTTP, rotas, proxy de bandeiras)
cmd/arenaget/       ferramenta de consulta à API do Arena
internal/arena/     cliente REST/OAuth do Arena
internal/mercure/   publicação/assinatura SSE e JWT
internal/live/      estado por tapete, sincronização, automações, controles, configurações
internal/hubproxy/  proxy /hub para os overlays (só tópicos cbw/)
internal/webhook/   recepção dos webhooks do Arena
web/                overlay, controle, configurações, galeria, lista
assets/             logo e bandeiras dos estados
scripts/            setup-env.sh
```

### API do cbw-live
| Método | Rota | Descrição |
|---|---|---|
| GET | `/api/state` | Estado de todos os tapetes |
| GET | `/api/state/{evento}/{tapete}` | Estado de um tapete |
| POST | `/api/control/{evento}/{tapete}` | Aciona um gráfico: `{"graphic":"challenge","visible":true,"side":"home","status":"won"}` |
| GET/PUT | `/api/settings` | Configurações globais |
| POST | `/webhook/arena` | Webhooks do Arena |
| GET | `/hub?topic=cbw/...` | Assinatura SSE (proxy do Mercure) |
| GET | `/flags/{código}.svg` | Bandeira (estados locais ou países do Arena) |
| GET | `/healthz` | Saúde do serviço |

`graphic`: `intro`, `challenge` (`status`: requested, challenge, won, lost), `injury` (`kind`: blood, injury), `passivity`, `victory` (`victoryType`), `layout` (`status`: expanded, collapsed) ou `all` (só para esconder). `side`: `home` (vermelho) ou `away` (azul).

## Segurança
- **Nunca** versione o `.env`. Ele contém o Secret do app, a API Key e a chave do Mercure.
- Se credenciais forem expostas (prints, chat), **gere novas**: Settings → Apps (novo Secret) e Settings → Users → (Re)generate API key.
- A tela **Settings → Users** do Arena mostra as API Keys de todos os usuários em texto. Restrinja o acesso de administrador.
- O cbw-live não tem autenticação: rode-o **apenas na rede local da produção**. Qualquer pessoa na rede pode abrir a página de controle.
- A instalação Docker do Arena expõe o Redis em `0.0.0.0:16370`. Em VPS/rede aberta, feche essa porta no firewall.
- O token do webhook ainda não é exigido (`WEBHOOK_TOKEN_ENFORCE=false`), porque o Arena não mostrou onde o envia no payload. Isso precisa ser revisto antes de expor o serviço fora da rede local.

## Limitações conhecidas e próximos passos
- **Challenge pedido/em análise, atendimento e passividade** são manuais. O LGTV oficial provavelmente recebe esses estados **direto do UWW Timing**. O próximo passo é capturar esse tráfego (Wireshark no Windows com o LGTV conectado) para automatizá-los.
- **Período** ("Period 1/2") é calculado a partir do tempo total enviado pelo Timing. O comportamento no intervalo ainda não foi confirmado.
- **Sem foto** dos atletas, porque o Arena não fornece.
- Os eventos de teste estão com fuso **Europe/Zurich** no Arena. Ajuste para **America/Sao_Paulo** para o "Início em" ficar correto.
