package live

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Gráficos que o operador pode acionar. Enquanto o UWW Timing não publicar
// esses estados (ainda não observados no Mercure), eles vêm da página de controle.
var graphics = map[string]bool{
	"intro":     true, // apresentação da luta (match ready)
	"challenge": true, // challenge: requested | challenge | won | lost
	"injury":    true, // atendimento: blood | injury, com cronômetro
	"passivity": true, // período de atividade (30 s)
	"victory":   true, // vencedor + tipo de vitória
	"layout":    true, // placar: status auto | expanded | collapsed
}

// gráficos que pertencem a um atleta (exigem side)
var perSide = map[string]bool{"challenge": true, "injury": true, "passivity": true, "victory": true}

type ControlRequest struct {
	Graphic     string `json:"graphic"`               // um de graphics, ou "all" (só com visible=false)
	Visible     bool   `json:"visible"`               //
	Side        string `json:"side,omitempty"`        // home | away
	Status      string `json:"status,omitempty"`      // challenge: requested|challenge|won|lost; layout: expanded|collapsed
	Kind        string `json:"kind,omitempty"`        // injury: blood|injury
	DurationSec int    `json:"durationSec,omitempty"` // injury/passivity
	VictoryType string `json:"victoryType,omitempty"` // victory: nome do tipo (ex.: VICTORY BY FALL)
}

type controlState struct {
	ControlRequest
	StartedAt int64 `json:"startedAt,omitempty"` // ms (relógio do servidor) — início do cronômetro
	UpdatedAt int64 `json:"updatedAt"`
	// Auto: acionado pelos dados do Timing/Arena (não pelo operador).
	Auto bool `json:"auto,omitempty"`
}

func (r ControlRequest) validate() error {
	if r.Graphic == "all" {
		if r.Visible {
			return fmt.Errorf(`"all" só pode ser usado para esconder`)
		}
		return nil
	}
	if !graphics[r.Graphic] {
		return fmt.Errorf("gráfico desconhecido: %q", r.Graphic)
	}
	if r.Visible && perSide[r.Graphic] && r.Side != "home" && r.Side != "away" {
		return fmt.Errorf(`side deve ser "home" ou "away"`)
	}
	if r.Graphic == "layout" && r.Visible && r.Status != "expanded" && r.Status != "collapsed" {
		return fmt.Errorf(`layout: status deve ser "expanded" ou "collapsed" (esconder = automático)`)
	}
	if r.DurationSec < 0 || r.DurationSec > 3600 {
		return fmt.Errorf("durationSec fora do intervalo")
	}
	return nil
}

// ApplyControl grava o estado do gráfico e o publica para os overlays do tapete.
func (s *Syncer) ApplyControl(ctx context.Context, eventID, matName string, req ControlRequest) (MatState, error) {
	if err := req.validate(); err != nil {
		return MatState{}, err
	}
	key := Key(eventID, matName)
	if _, ok := s.Store.Get(key); !ok {
		return MatState{}, fmt.Errorf("tapete não encontrado: %s", key)
	}
	now := time.Now().UnixMilli()
	state, _ := s.Store.Update(key, func(m *MatState) bool {
		if req.Graphic == "all" {
			clearGraphics(m)
			return true
		}
		if !req.Visible {
			delete(m.Control, req.Graphic)
			return true
		}
		cs := controlState{ControlRequest: req, UpdatedAt: now}
		// o cronômetro só reinicia se for um novo acionamento (não ao trocar status/lado)
		if prev, ok := m.Control[req.Graphic]; ok {
			var p controlState
			if json.Unmarshal(prev, &p) == nil && p.Side == req.Side {
				cs.StartedAt = p.StartedAt
			}
		}
		if cs.StartedAt == 0 {
			cs.StartedAt = now
		}
		b, _ := json.Marshal(cs)
		m.Control[req.Graphic] = b
		return true
	})
	s.publish(ctx, eventID, matName, message{Type: "control", State: &state})
	return state, nil
}

// clearGraphics remove os gráficos no ar, preservando a preferência de layout do operador.
func clearGraphics(m *MatState) {
	for k := range m.Control {
		if k != "layout" {
			delete(m.Control, k)
		}
	}
}
