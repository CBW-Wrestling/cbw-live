package live

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"cbwlive/internal/timing"
)

// Conexão direta com o UWW Timing (WebSocket do "UWW TV Scoreboard", porta 9696).
// É a fonte mais completa: além de placar e tempo, traz pausa do cronômetro,
// challenge em cada etapa, atendimento (sangue), passividade, match ready e vitória.
// Enquanto o Timing de um tapete está conectado, o tópico Mercure vira reserva.

// timingFeed acompanha um PC de UWW Timing. log (opcional) grava cada mensagem crua.
type timingFeed struct {
	s    *Syncer
	addr string
	log  *Recorder

	mu        sync.Mutex
	key       string // evento/tapete ao qual este Timing está ligado
	eventID   string
	matName   string
	fightID   string
	matchNo   int
	round     string
	category  string
	clock     string
	running   bool
	started   bool
	advantage string
	home      timingAthlete
	away      timingAthlete
}

type timingAthlete struct {
	Name     string `json:"name"`
	Country  string `json:"country"`
	Score    int    `json:"score"`
	Cautions int    `json:"cautions"`
	Active   bool   `json:"active"`
}

// RunTimingFeed mantém a conexão com um UWW Timing até ctx terminar.
func (s *Syncer) RunTimingFeed(ctx context.Context, addr string, log *Recorder) {
	f := &timingFeed{s: s, addr: addr, log: log}
	timing.Listen(ctx, addr, func(msg string) { f.handle(ctx, msg) }, func(connected bool) {
		if connected {
			go f.retryResolve(ctx)
		} else {
			f.unbind()
		}
	})
	f.unbind()
}

func (s *Syncer) timingActive(key string) bool {
	s.tmu.Lock()
	defer s.tmu.Unlock()
	return s.timingKeys[key] > 0
}

func (s *Syncer) timingActiveForFight(fightID string) bool {
	for _, m := range s.Store.All() {
		if m.LiveFightID == fightID || fightIDOf(m.Fight) == fightID {
			return s.timingActive(Key(m.EventID, m.MatName))
		}
	}
	return false
}

func fightIDOf(raw json.RawMessage) string {
	var f struct {
		ID string `json:"id"`
	}
	json.Unmarshal(raw, &f)
	return f.ID
}

// bind liga este Timing a um tapete (evento + nome do tapete).
func (f *timingFeed) bind(eventID, matName string) {
	key := Key(eventID, matName)
	if key == f.key {
		return
	}
	f.unbindLocked()
	f.key, f.eventID, f.matName = key, eventID, matName
	f.s.tmu.Lock()
	f.s.timingKeys[key]++
	f.s.tmu.Unlock()
	f.s.Store.Update(key, func(m *MatState) bool { m.TimingAddr = f.addr; return true })
	slog.Info("timing: ligado ao tapete", "addr", f.addr, "tapete", matName, "evento", eventID)
}

func (f *timingFeed) unbind() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unbindLocked()
}

func (f *timingFeed) unbindLocked() {
	if f.key == "" {
		return
	}
	f.s.tmu.Lock()
	if f.s.timingKeys[f.key]--; f.s.timingKeys[f.key] <= 0 {
		delete(f.s.timingKeys, f.key)
	}
	f.s.tmu.Unlock()
	f.s.Store.Update(f.key, func(m *MatState) bool {
		if m.TimingAddr == f.addr {
			m.TimingAddr = ""
		}
		return true
	})
	slog.Info("timing: desligado do tapete", "addr", f.addr, "tapete", f.matName)
	f.key = ""
}

// resolveByNames liga o Timing ao tapete que tem uma luta aberta com os mesmos
// atletas — usado quando o cbw-live (re)conecta no meio de uma luta, antes de
// qualquer "mch" (o estado completo do Timing não traz evento nem tapete).
func (f *timingFeed) resolveByNames() bool {
	if f.key != "" {
		return true
	}
	if f.home.Name == "" {
		return false
	}
	norm := func(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
	for _, m := range f.s.Store.All() {
		for _, raw := range m.openFights {
			var fight struct {
				ID string `json:"id"`
				N1 string `json:"fighter1DisplayName"`
				N2 string `json:"fighter2DisplayName"`
			}
			if json.Unmarshal(raw, &fight) != nil || fight.N1 == "" {
				continue
			}
			if norm(fight.N1) == norm(f.home.Name) && norm(fight.N2) == norm(f.away.Name) {
				f.fightID = fight.ID
				f.bind(m.EventID, m.MatName)
				return true
			}
		}
	}
	return false
}

// retryResolve tenta identificar o tapete por alguns segundos depois de conectar
// (os tapetes do Arena podem ainda não ter sido carregados quando o estado chega).
func (f *timingFeed) retryResolve(ctx context.Context) {
	for i := 0; i < 20; i++ {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
		f.mu.Lock()
		ok := f.resolveByNames()
		if ok && f.key != "" {
			f.s.ingestLive(ctx, f.eventID, f.matName, f.liveJSON(), time.Now())
		}
		f.mu.Unlock()
		if ok {
			return
		}
	}
	slog.Warn("timing: não identifiquei o tapete; ele será ligado quando uma luta for carregada no Timing", "addr", f.addr)
}

func (f *timingFeed) handle(ctx context.Context, raw string) {
	if f.log != nil {
		f.log.Write(map[string]any{"addr": f.addr, "msg": raw})
	}
	msg := timing.Parse(raw)

	f.mu.Lock()
	defer f.mu.Unlock()

	publishLive := false
	switch msg.Head {
	case "state":
		st := msg.State
		f.matchNo, _ = strconv.Atoi(st.MatchNo)
		f.category, f.round, f.clock = st.Category, st.Round, st.Clock
		f.home = timingAthlete{Name: st.HomeName, Country: st.HomeCountry, Score: st.HomeScore, Cautions: st.HomeCautions, Active: true}
		f.away = timingAthlete{Name: st.AwayName, Country: st.AwayCountry, Score: st.AwayScore, Cautions: st.AwayCautions, Active: true}
		f.advantage = advSide(strconv.Itoa(st.Advantage))
		f.started = f.started || st.Clock != ""
		f.resolveByNames()
		publishLive = true

	case "mch": // luta carregada: 9;Qualif.;Seniors FS 97;A;cores×4;fightId;Seniors;2;180;cntDown;wcId;eventId;eventName
		f.matchNo, _ = strconv.Atoi(msg.Arg(0))
		f.round, f.category = msg.Arg(1), msg.Arg(2)
		f.fightID = msg.Arg(8)
		f.home.Score, f.away.Score, f.home.Cautions, f.away.Cautions = 0, 0, 0, 0
		f.advantage, f.running, f.started = "", false, false
		if ev, mat := msg.Arg(14), msg.Arg(3); ev != "" && mat != "" {
			f.bind(ev, mat)
		}
		publishLive = true

	case "wt1", "wt2": // wt1;NOME(PAÍS);wt2;NOME(PAÍS)
		if msg.Head == "wt1" {
			f.home.Name, f.home.Country = timing.SplitNameCountry(msg.Arg(0))
		} else {
			f.away.Name, f.away.Country = timing.SplitNameCountry(msg.Arg(0))
		}
		for i, a := range msg.Args {
			if a == "wt2" {
				f.away.Name, f.away.Country = timing.SplitNameCountry(msg.Arg(i + 1))
			}
		}
		f.home.Active, f.away.Active = true, true
		publishLive = true

	case "sc1":
		if x, y, ok := msg.Pair(); ok {
			f.home.Score, f.away.Score = x, y
			publishLive = true
		}
	case "ct1":
		if x, y, ok := msg.Pair(); ok {
			f.home.Cautions, f.away.Cautions = x, y
			publishLive = true
		}
	case "avt":
		f.advantage = advSide(msg.Arg(0))
		publishLive = true

	case "clk": // clk;T[;start|stop|stopend]
		if t := msg.Arg(0); t != "" {
			f.clock = t
		}
		switch msg.Arg(1) {
		case "start":
			f.running, f.started = true, true
			f.control(ctx, func(m *MatState, now time.Time) bool { return removeAuto(m, "intro") })
		case "stop", "stopend":
			f.running = false
		}
		publishLive = true

	case "pre": // match ready
		f.started, f.running = false, false
		if f.s.Settings == nil || f.s.Settings.Get().AutoIntro {
			fight := f.fightID
			f.control(ctx, func(m *MatState, now time.Time) bool {
				if m.introFor == fight && fight != "" {
					return false
				}
				m.introFor = fight
				setControl(m, ControlRequest{Graphic: "intro", Visible: true}, now)
				return true
			})
		}
		publishLive = true
	case "rdy":
		f.control(ctx, func(m *MatState, now time.Time) bool { return removeAuto(m, "intro") })

	case "ch1", "ch2":
		side, status := msg.Side(), msg.ChallengeStatus()
		if side == "" || status == "" {
			break
		}
		var stamp int64
		f.control(ctx, func(m *MatState, now time.Time) bool {
			setControl(m, ControlRequest{Graphic: "challenge", Visible: true, Side: side, Status: status}, now)
			stamp = now.UnixMilli()
			return true
		})
		if status == "won" || status == "lost" || status == "canceled" {
			f.s.hideLater(f.eventID, f.matName, "challenge", stamp, ChallengeHold)
		}

	case "ij1", "ij2", "ac1", "ac2": // T;[start|show|stop|hide|stopend] — sem comando = tique do relógio
		graphic := "injury"
		if strings.HasPrefix(msg.Head, "ac") {
			graphic = "passivity"
		}
		side, t, cmd := msg.Side(), msg.Arg(0), strings.ToLower(msg.Arg(1))
		f.control(ctx, func(m *MatState, now time.Time) bool {
			if cmd == "hide" || cmd == "stopend" {
				return removeAuto(m, graphic)
			}
			req := ControlRequest{Graphic: graphic, Visible: true, Side: side}
			if graphic == "injury" {
				req.Kind = "blood"
			}
			cs := controlState{ControlRequest: req, UpdatedAt: now.UnixMilli(), StartedAt: now.UnixMilli(),
				Auto: true, Time: t, Paused: cmd == "stop"}
			if prev, ok := controlOf(m, graphic); ok && prev.Side == side {
				cs.StartedAt = prev.StartedAt
				if cmd == "" { // tique: mantém o estado de pausa anterior só se não estava pausado
					cs.Paused = false
				}
			}
			b, _ := json.Marshal(cs)
			m.Control[graphic] = b
			return true
		})

	case "wmh": // wmh;VENCEDOR[;5-0VFA]
		name, code := msg.Arg(0), timing.VictoryCode(msg.Arg(1))
		if code == "" {
			break // só o nome: a mesa ainda está escolhendo o tipo
		}
		side := ""
		switch strings.ToUpper(strings.TrimSpace(name)) {
		case strings.ToUpper(f.home.Name):
			side = "home"
		case strings.ToUpper(f.away.Name):
			side = "away"
		}
		if side == "" {
			break
		}
		f.running = false
		f.control(ctx, func(m *MatState, now time.Time) bool {
			removeAuto(m, "challenge")
			removeAuto(m, "injury")
			removeAuto(m, "passivity")
			setControl(m, ControlRequest{Graphic: "victory", Visible: true, Side: side, VictoryType: victoryName(m.Fight, code)}, now)
			return true
		})
		publishLive = true
	}

	if publishLive && f.key != "" {
		f.s.ingestLive(ctx, f.eventID, f.matName, f.liveJSON(), time.Now())
	}
}

// control aplica uma mudança de gráfico no tapete ligado e publica.
func (f *timingFeed) control(ctx context.Context, fn func(m *MatState, now time.Time) bool) {
	if f.key == "" {
		return
	}
	now := time.Now()
	state, changed := f.s.Store.Update(f.key, func(m *MatState) bool { return fn(m, now) })
	if changed {
		f.s.publish(ctx, f.eventID, f.matName, message{Type: "control", State: &state})
	}
}

// liveJSON monta a mensagem no mesmo formato que o Timing publica no Mercure,
// para overlays e automações não precisarem saber de onde veio.
func (f *timingFeed) liveJSON() json.RawMessage {
	state := "starting"
	if f.started {
		state = "running"
	}
	running := f.running
	b, _ := json.Marshal(map[string]any{
		"sportEventId": f.eventID, "matName": f.matName, "state": state,
		"source": "timing", "clockRunning": &running,
		"message": map[string]any{
			"id": f.fightID, "idle": false, "weightCategory": f.category, "round": f.round,
			"matchNo": f.matchNo, "time": f.clock, "advantage": f.advantage,
			"home": f.home, "away": f.away,
		},
	})
	return b
}

func advSide(v string) string {
	switch strings.TrimSpace(v) {
	case "1":
		return "home"
	case "2":
		return "away"
	}
	return ""
}

// removeAuto tira um gráfico do ar se ele foi acionado automaticamente.
func removeAuto(m *MatState, graphic string) bool {
	if cs, ok := controlOf(m, graphic); ok && cs.Auto {
		delete(m.Control, graphic)
		return true
	}
	return false
}

// hideLater tira o gráfico depois de d, se ninguém o alterou nesse meio-tempo.
func (s *Syncer) hideLater(eventID, matName, graphic string, stamp int64, d time.Duration) {
	key := Key(eventID, matName)
	time.AfterFunc(d, func() {
		st, changed := s.Store.Update(key, func(m *MatState) bool {
			cs, ok := controlOf(m, graphic)
			if !ok || !cs.Auto || cs.UpdatedAt != stamp {
				return false
			}
			delete(m.Control, graphic)
			return true
		})
		if changed {
			s.publish(context.Background(), eventID, matName, message{Type: "control", State: &st})
		}
	})
}
