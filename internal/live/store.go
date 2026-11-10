// Package live mantém o estado atual de cada tapete em memória e o publica
// no Mercure sempre que muda.
package live

import (
	"encoding/json"
	"maps"
	"sort"
	"sync"
	"time"
)

type Colors struct {
	Athlete1     string `json:"athlete1"`
	Athlete1Text string `json:"athlete1Text"`
	Athlete2     string `json:"athlete2"`
	Athlete2Text string `json:"athlete2Text"`
}

type MatState struct {
	EventID   string `json:"eventId"`
	EventName string `json:"eventName"`
	EventCity string `json:"eventCity"`
	MatID     string `json:"matId"`
	MatName   string `json:"matName"`
	Colors    Colors `json:"colors"`

	// Luta atual do tapete (a carregada no placar oficial; senão a primeira aberta), formato bruto do Arena.
	Fight      json.RawMessage `json:"fight"`
	OpenFights int             `json:"openFights"`

	// Última mensagem que o placar oficial publicou no Mercure para este tapete.
	Live   json.RawMessage `json:"live,omitempty"`
	LiveAt *time.Time      `json:"liveAt,omitempty"`
	// ID da luta carregada no placar oficial (message.id da mensagem ao vivo).
	LiveFightID string `json:"liveFightId,omitempty"`

	// Gráficos acionados pelo operador (intro, challenge, injury, passivity, victory).
	Control map[string]json.RawMessage `json:"control"`

	UpdatedAt time.Time `json:"updatedAt"`
	Seq       uint64    `json:"seq"`
	// luta para a qual a apresentação automática já foi exibida (não reapresenta)
	introFor string
	// Relógio do servidor (ms) no momento do envio — os overlays usam para
	// sincronizar cronômetros de atendimento/passividade sem depender do relógio da máquina.
	ServerTime int64 `json:"serverTime"`
}

// clone devolve uma cópia sem compartilhar o mapa de controles.
func (m *MatState) clone() MatState {
	c := *m
	c.Control = maps.Clone(m.Control)
	if c.Control == nil {
		c.Control = map[string]json.RawMessage{}
	}
	c.ServerTime = time.Now().UnixMilli()
	return c
}

func Key(eventID, matName string) string { return eventID + "/" + matName }

// Topic é o tópico Mercure que os overlays da CBW assinam.
func Topic(eventID, matName string) string { return "cbw/live/" + eventID + "/" + matName }

type Store struct {
	mu   sync.RWMutex
	mats map[string]*MatState
}

func NewStore() *Store { return &Store{mats: map[string]*MatState{}} }

// Update aplica fn ao estado do tapete (criando se preciso) e devolve uma cópia
// do resultado e se houve mudança.
func (s *Store) Update(key string, fn func(m *MatState) bool) (MatState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.mats[key]
	if !ok {
		m = &MatState{Fight: json.RawMessage("null"), Control: map[string]json.RawMessage{}}
		s.mats[key] = m
	}
	changed := fn(m)
	if changed {
		m.Seq++
		m.UpdatedAt = time.Now()
	}
	return m.clone(), changed
}

func (s *Store) Get(key string) (MatState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.mats[key]
	if !ok {
		return MatState{}, false
	}
	return m.clone(), true
}

func (s *Store) All() []MatState {
	s.mu.RLock()
	out := make([]MatState, 0, len(s.mats))
	for _, m := range s.mats {
		out = append(out, m.clone())
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].EventName != out[j].EventName {
			return out[i].EventName < out[j].EventName
		}
		return out[i].MatName < out[j].MatName
	})
	return out
}
