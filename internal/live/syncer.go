package live

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"cbwlive/internal/arena"
	"cbwlive/internal/mercure"
)

// uwwLiveTopic é o tópico em que o placar oficial (UWW Timing) publica o
// estado ao vivo de cada tapete — descoberto no publish_token de
// /api/json/scoreboard/mercuretoken.
func uwwLiveTopic(eventID, matName string) string {
	return "http://arena.uww.org/live/" + eventID + "/" + matName
}

type Syncer struct {
	Arena   *arena.Client
	Hub     *mercure.Hub
	Store   *Store
	EventID string // vazio = todos
	Resync  time.Duration
	LiveLog *Recorder
	// Settings (opcional) controla automações como a apresentação automática.
	Settings *SettingsStore

	trigger chan struct{}
	// tapetes com o UWW Timing conectado direto (contagem por tapete)
	tmu        sync.Mutex
	timingKeys map[string]int
	mu         sync.Mutex
	mats       map[string]arena.Mat // key -> mat
	subs       map[string]context.CancelFunc
}

type message struct {
	Type  string    `json:"type"` // "snapshot" | "live"
	State *MatState `json:"state,omitempty"`
	Live  any       `json:"live,omitempty"`
}

func NewSyncer(a *arena.Client, hub *mercure.Hub, store *Store, eventID string, resync time.Duration, liveLog *Recorder) *Syncer {
	return &Syncer{
		Arena: a, Hub: hub, Store: store, EventID: eventID, Resync: resync, LiveLog: liveLog,
		trigger:    make(chan struct{}, 1),
		timingKeys: map[string]int{},
		mats:       map[string]arena.Mat{},
		subs:       map[string]context.CancelFunc{},
	}
}

// Trigger pede uma sincronização (com debounce). Não bloqueia.
func (s *Syncer) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}

// SniffAll assina todos os tópicos do hub e grava cada mensagem — usado na fase
// de descoberta para ver o que o próprio Arena publica (ex.: arena-mq-processor).
func (s *Syncer) SniffAll(ctx context.Context, rec *Recorder) {
	s.Hub.Subscribe(ctx, []string{"*"}, func(e mercure.Event) {
		rec.Write(map[string]any{"id": e.ID, "type": e.Type, "data": RawOrString(e.Data)})
	})
}

func (s *Syncer) Run(ctx context.Context) {
	s.sync(ctx)

	var tick <-chan time.Time
	if s.Resync > 0 {
		t := time.NewTicker(s.Resync)
		defer t.Stop()
		tick = t.C
	}
	const debounce = 150 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.trigger:
			// junta rajadas de webhooks (um ponto gera vários eventos de entidade)
			timer := time.NewTimer(debounce)
		drain:
			for {
				select {
				case <-s.trigger:
				case <-timer.C:
					break drain
				case <-ctx.Done():
					timer.Stop()
					return
				}
			}
			s.sync(ctx)
		case <-tick:
			s.sync(ctx)
		}
	}
}

func (s *Syncer) sync(ctx context.Context) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	events, err := s.Arena.ScoreboardEvents(ctx)
	if err != nil {
		slog.Error("sync: falha ao listar eventos", "err", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, ev := range events {
		if s.EventID != "" && ev.ID != s.EventID {
			continue
		}
		for _, mat := range ev.Mats {
			ev, mat := ev, mat
			s.ensureLiveSub(ctx, ev.ID, mat)
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer func() { <-sem; wg.Done() }()
				s.refreshMat(ctx, ev, mat)
			}()
		}
	}
	wg.Wait()
	slog.Debug("sync concluído", "em", time.Since(start))
}

func (s *Syncer) refreshMat(ctx context.Context, ev arena.SportEvent, mat arena.Mat) {
	fights, err := s.Arena.OpenFights(ctx, mat.ID)
	if err != nil {
		slog.Warn("sync: falha ao buscar luta aberta", "mat", mat.Name, "err", err)
		return
	}
	key := Key(ev.ID, mat.Name)
	current, _ := s.Store.Get(key)
	fight := pickFight(fights, current.LiveFightID)
	state, changed := s.Store.Update(key, func(m *MatState) bool {
		meta := m.EventID != ev.ID || m.MatID != mat.ID || m.EventName != ev.Name || m.EventCity != ev.AddressLocality
		m.EventID, m.EventName, m.EventCity, m.MatID, m.MatName = ev.ID, ev.Name, ev.AddressLocality, mat.ID, mat.Name
		m.Colors = Colors{ev.Athlete1Color, ev.Athlete1TextColor, ev.Athlete2Color, ev.Athlete2TextColor}
		fightChanged := !bytes.Equal(m.Fight, fight) || m.OpenFights != len(fights)
		m.Fight, m.OpenFights, m.openFights = fight, len(fights), fights
		return meta || fightChanged
	})
	if changed {
		s.publish(ctx, state.EventID, state.MatName, message{Type: "snapshot", State: &state})
		slog.Info("tapete atualizado", "evento", ev.Name, "mat", mat.Name, "lutasAbertas", len(fights), "seq", state.Seq)
	}
}

// pickFight escolhe a luta carregada no placar oficial; sem essa informação,
// a primeira luta aberta do tapete.
func pickFight(fights []json.RawMessage, liveFightID string) json.RawMessage {
	if len(fights) == 0 {
		return json.RawMessage("null")
	}
	if liveFightID != "" {
		for _, f := range fights {
			var probe struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(f, &probe) == nil && probe.ID == liveFightID {
				return f
			}
		}
	}
	return fights[0]
}

// liveMessage é o formato publicado pelo UWW Timing em http://arena.uww.org/live/{evento}/{tapete}.
// detail só aparece no fim da luta (medido em 09/11/2026):
//
//	"detail": {"winner": "<code do atleta>", "victory": "VFA", "victoryTime": "00:26"}
//
// winner chega primeiro com victory vazio; o tipo vem quando a mesa confirma.
type liveMessage struct {
	State string `json:"state"` // "starting" (luta carregada) | "running"
	// Source: "timing" quando montada pelo cbw-live a partir do WebSocket do Timing
	// (aí apresentação/vitória já são tratadas pelo timingFeed).
	Source  string `json:"source"`
	Message struct {
		ID     string                `json:"id"`
		Idle   bool                  `json:"idle"`
		Home   struct{ Code string } `json:"home"`
		Away   struct{ Code string } `json:"away"`
		Detail *struct {
			Winner      string `json:"winner"`
			Victory     string `json:"victory"`
			VictoryTime string `json:"victoryTime"`
		} `json:"detail"`
	} `json:"message"`
}

// ensureLiveSub assina (uma vez) o tópico do placar oficial para este tapete.
func (s *Syncer) ensureLiveSub(ctx context.Context, eventID string, mat arena.Mat) {
	key := Key(eventID, mat.Name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subs[key]; ok {
		return
	}
	subCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.subs[key] = cancel
	topic := uwwLiveTopic(eventID, mat.Name)
	go s.Hub.Subscribe(subCtx, []string{topic}, func(e mercure.Event) {
		now := time.Now()
		if s.LiveLog != nil {
			s.LiveLog.Write(map[string]any{"topic": topic, "id": e.ID, "type": e.Type, "data": RawOrString(e.Data)})
		}
		data := json.RawMessage(e.Data)
		if !json.Valid(data) {
			b, _ := json.Marshal(string(e.Data))
			data = b
		}
		// com o Timing conectado direto (fonte mais completa), o Mercure vira reserva
		if s.timingActive(key) {
			return
		}
		s.ingestLive(subCtx, eventID, mat.Name, data, now)
	})
}

// ingestLive aplica uma mensagem ao vivo no formato do Timing/Mercure — vinda do
// tópico Mercure ou montada a partir do WebSocket do Timing — e publica.
func (s *Syncer) ingestLive(ctx context.Context, eventID, matName string, data json.RawMessage, now time.Time) {
	key := Key(eventID, matName)
	var lm liveMessage
	json.Unmarshal(data, &lm)
	fightChanged := false
	state, _ := s.Store.Update(key, func(m *MatState) bool {
		m.Live, m.LiveAt = data, &now
		if lm.Message.ID != "" && lm.Message.ID != m.LiveFightID {
			// nova luta carregada: gráficos da luta anterior (challenge, vitória…) saem
			m.LiveFightID, fightChanged = lm.Message.ID, m.LiveFightID != ""
			clearGraphics(m)
		}
		if s.autoFromLive(m, lm, now) {
			fightChanged = true // força publicar os controles
		}
		return true
	})
	s.publish(ctx, eventID, matName, message{Type: "live", Live: data})
	if fightChanged {
		s.publish(ctx, eventID, matName, message{Type: "control", State: &state})
		// atualiza o snapshot (atletas, categoria, cores)
		s.Trigger()
	}
}

func (s *Syncer) publish(ctx context.Context, eventID, matName string, msg message) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := s.Hub.Publish(ctx, Topic(eventID, matName), b); err != nil {
		slog.Warn("mercure: falha ao publicar", "mat", matName, "err", err)
	}
}

// Close encerra as assinaturas do placar oficial.
func (s *Syncer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.subs {
		c()
	}
}
